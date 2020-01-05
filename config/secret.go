package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

// SecretType represents the type of secret.
type SecretType string

const (
	// DataSecret represents secret that contains value in Data field.
	DataSecret SecretType = "Data"
	// FileSecret represents secret that contains value in file.
	FileSecret SecretType = "File"
	// EnvSecret represents secret that contains
	// value in environment variable.
	EnvSecret SecretType = "Env"
)

// Secret stores configuration for secret data.
//
// Used for inserting secret values to configs like passwords and tokens.
// If you want to pass secret as plain text, use type DataSecret:
//   Secret{Type: DataSecret, Data: "qwerty123"}
// For loading secret from file you should use FileSecret type:
//   Secret{Type: FileSecret, Data: "db-password.txt"}
// For passing environment variable to secret you should use EnvSecret:
//   Secret{Type: EnvSecret, Data: "DB_PASSWORD"}
type Secret struct {
	// Type contains secret type.
	Type SecretType `json:""`
	// Data contains secret data.
	Data string `json:""`
	// cache contains cached secret value.
	cache atomic.Value
}

// Secret returns secret value.
func (s *Secret) Secret() (string, error) {
	if data := s.cache.Load(); data != nil {
		return data.(string), nil
	}
	return s.secretLocked()
}

var mutex sync.Mutex

// secretLocked returns secret value with locking mutex.
func (s *Secret) secretLocked() (string, error) {
	mutex.Lock()
	defer mutex.Unlock()
	// Recheck that cache is empty. This action is
	// required due to race conditions.
	if data := s.cache.Load(); data != nil {
		return data.(string), nil
	}
	switch s.Type {
	case FileSecret:
		bytes, err := ioutil.ReadFile(s.Data)
		if err != nil {
			return "", err
		}
		s.cache.Store(strings.TrimRight(string(bytes), "\r\n"))
	case EnvSecret:
		value, ok := os.LookupEnv(s.Data)
		if !ok {
			return "", fmt.Errorf(
				"environment variable %q does not exists", s.Data,
			)
		}
		s.cache.Store(value)
	case DataSecret:
		s.cache.Store(s.Data)
	default:
		return "", fmt.Errorf("unsupported secret type %q", s.Type)
	}
	return s.cache.Load().(string), nil
}

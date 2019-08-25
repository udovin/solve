package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

type SecretType string

const (
	DataSecret SecretType = "Data"
	FileSecret SecretType = "File"
	EnvSecret  SecretType = "Env"
)

var mutex sync.Mutex

// Secret stores configuration for secret data
//
// Used for inserting secret values to configs like passwords and tokens.
// If you want to pass secret as plain text, use type DataSecret:
//   Secret{Type: DataSecret, Data: "qwerty123"}
// For loading secret from file you should use FileSecret type:
//   Secret{Type: FileSecret, Data: "db-password.txt"}
// For passing environment variable to secret you should use EnvSecret:
//   Secret{Type: EnvSecret, Data: "DB_PASSWORD"}
type Secret struct {
	Type SecretType `json:""`
	Data string     `json:""`
}

// GetValue returns secret value
func (s *Secret) GetValue() (string, error) {
	mutex.Lock()
	switch s.Type {
	case FileSecret:
		bytes, err := ioutil.ReadFile(s.Data)
		if err != nil {
			mutex.Unlock()
			return "", err
		}
		s.Data = strings.TrimRight(string(bytes), "\r\n")
		s.Type = DataSecret
	case EnvSecret:
		value, ok := os.LookupEnv(s.Data)
		if !ok {
			mutex.Unlock()
			return "", fmt.Errorf(
				"environment variable '%s' does not exists", s.Data,
			)
		}
		s.Data, s.Type = value, DataSecret
	}
	mutex.Unlock()
	if s.Type == DataSecret {
		return s.Data, nil
	}
	return "", fmt.Errorf(
		"unsupported secret type '%s'", s.Type,
	)
}

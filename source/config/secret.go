package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type SecretType string

const (
	ValueSecret    SecretType = "Value"
	VariableSecret SecretType = "Variable"
	FileSecret     SecretType = "File"
)

type Secret struct {
	Type SecretType `json:""`
	Data string     `json:""`
}

func (s *Secret) GetValue() (string, error) {
	switch s.Type {
	case ValueSecret:
		return s.Data, nil
	case VariableSecret:
		value, ok := os.LookupEnv(s.Data)
		if !ok {
			return "", fmt.Errorf(
				"environment variable '%s' does not exists",
				s.Data,
			)
		}
		return value, nil
	case FileSecret:
		bytes, err := ioutil.ReadFile(s.Data)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(bytes), "\r\n"), nil
	}
	return "", fmt.Errorf("unsupported secret type '%s'", s.Type)
}

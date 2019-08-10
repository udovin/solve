package config

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestSecret_GetValue_DataSecret(t *testing.T) {
	expectedValue := "Hello, World!"
	s := Secret{Type: DataSecret, Data: expectedValue}
	value, err := s.GetValue()
	if err != nil {
		t.Error("Error: ", err)
	}
	if value != expectedValue {
		t.Errorf(
			"Expected '%s', but got '%s'",
			expectedValue, value,
		)
	}
}

func TestSecret_GetValue_FileSecret(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "solve-test-")
	if err != nil {
		t.Error("Error: ", err)
	}
	expectedValue := "Hello, World!"
	_, err = file.Write([]byte(expectedValue))
	_ = file.Close()
	defer func() {
		_ = os.Remove(file.Name())
	}()
	if err != nil {
		t.Error("Error: ", err)
	}
	s := Secret{Type: FileSecret, Data: file.Name()}
	value, err := s.GetValue()
	if err != nil {
		t.Error("Error: ", err)
	}
	if value != expectedValue {
		t.Errorf(
			"Expected '%s', but got '%s'",
			expectedValue, value,
		)
	}
	s = Secret{Type: FileSecret, Data: s.Data + "-invalid"}
	if _, err := s.GetValue(); err == nil {
		t.Error("Expected error")
	}
}

func TestSecret_GetValue_EnvSecret(t *testing.T) {
	name := "SOLVE_TEST_ENV_VAR"
	expectedValue := "Hello, World!"
	err := os.Setenv(name, expectedValue)
	if err != nil {
		t.Error("Error: ", err)
	}
	s := Secret{Type: EnvSecret, Data: name}
	value, err := s.GetValue()
	if err != nil {
		t.Error("Error: ", err)
	}
	if value != expectedValue {
		t.Errorf(
			"Expected '%s', but got '%s'",
			expectedValue, value,
		)
	}
	s = Secret{Type: EnvSecret, Data: s.Data + "_INVALID"}
	if _, err := s.GetValue(); err == nil {
		t.Error("Expected error")
	}
}

func TestSecret_GetValue_Unsupported(t *testing.T) {
	s := Secret{Type: "Unsupported", Data: "12345"}
	_, err := s.GetValue()
	if err == nil {
		t.Error("Expected error")
	}
}

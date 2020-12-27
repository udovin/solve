package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/labstack/gommon/log"
)

func TestLoadFromFile(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "solve-test-")
	if err != nil {
		t.Error("Error: ", err)
	}
	expectedConfig := Config{
		Server: Server{
			Host: "localhost",
			Port: 4242,
		},
		DB: DB{
			Driver:  SQLiteDriver,
			Options: SQLiteOptions{Path: "?mode=memory"},
		},
		LogLevel: log.INFO,
	}
	expectedConfigData, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Fatal("Error: ", err)
	}
	_, err = file.Write(expectedConfigData)
	_ = file.Close()
	defer func() {
		_ = os.Remove(file.Name())
	}()
	if err != nil {
		t.Fatal("Error: ", err)
	}
	_, err = LoadFromFile(path.Join(os.TempDir(), "solve-test-deleted"))
	if err == nil {
		t.Fatal("Expected error for config from deleted file")
	}
	config, err := LoadFromFile(file.Name())
	if err != nil {
		t.Fatal("Error: ", err)
	}
	if config != expectedConfig {
		t.Fatal("Config was corrupted")
	}
}

func TestLoadFromInvalidFile(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "solve-test-")
	if err != nil {
		t.Error("Error: ", err)
	}
	_, err = file.Write([]byte("invalid data"))
	_ = file.Close()
	defer func() {
		_ = os.Remove(file.Name())
	}()
	if _, err := LoadFromFile(file.Name()); err == nil {
		t.Fatal("Expected error for invalid config file")
	}
}

func TestServerAddress(t *testing.T) {
	s := Server{Host: "localhost", Port: 8080}
	addr := "localhost:8080"
	if v := s.Address(); v != addr {
		t.Fatalf("Expected %q, got %q", addr, v)
	}
}

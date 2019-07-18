package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "solve-test-")
	if err != nil {
		t.Error("Error: ", err)
	}
	expectedConfig := Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 4242,
		},
		Database: DatabaseConfig{
			Driver:  SQLiteDriver,
			Options: SQLiteOptions{Path: "?mode=memory"},
		},
	}
	expectedConfigData, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Error("Error: ", err)
	}
	_, err = file.Write(expectedConfigData)
	_ = file.Close()
	defer func() {
		_ = os.Remove(file.Name())
	}()
	if err != nil {
		t.Error("Error: ", err)
	}
	_, err = LoadFromFile(path.Join(os.TempDir(), "solve-test-deleted"))
	if err == nil {
		t.Error("Expected error for config from deleted file")
	}
	config, err := LoadFromFile(file.Name())
	if config != expectedConfig {
		t.Error("Config was corrupted")
	}
}

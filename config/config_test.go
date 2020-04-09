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
		Server: Server{
			Host: "localhost",
			Port: 4242,
		},
		DB: DB{
			Driver:  SQLiteDriver,
			Options: SQLiteOptions{Path: "?mode=memory"},
		},
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

func TestServer_Address(t *testing.T) {
	s := Server{Host: "localhost", Port: 8080}
	addr := "localhost:8080"
	if v := s.Address(); v != addr {
		t.Fatalf("Expected %q, got %q", addr, v)
	}
}

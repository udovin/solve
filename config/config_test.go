package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/gommon/log"
)

func TestLoadFromFile(t *testing.T) {
	expectedConfig := Config{
		Server: &Server{
			Host: "localhost",
			Port: 4242,
		},
		DB: DB{
			Options: SQLiteOptions{Path: ":memory:"},
		},
		LogLevel: LogLevel(log.INFO),
	}
	expectedConfigData, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Fatal("Error: ", err)
	}
	file, err := os.CreateTemp(t.TempDir(), "solve-test-")
	if err != nil {
		t.Error("Error: ", err)
	}
	func() {
		defer func() { _ = file.Close() }()
		if _, err := file.Write(expectedConfigData); err != nil {
			t.Fatal("Error: ", err)
		}
	}()
	if _, err := LoadFromFile(filepath.Join(t.TempDir(), "solve-test-deleted")); err == nil {
		t.Fatal("Expected error for config from deleted file")
	}
	config, err := LoadFromFile(file.Name())
	if err != nil {
		t.Fatal("Error: ", err)
	}
	configData, err := json.Marshal(config)
	if err != nil {
		t.Fatal("Error: ", err)
	}
	testExpect(t, string(configData), string(expectedConfigData))
}

const templateConfig = `
{
	"server": {
		"host": {{ "localhost" | json }},
		"port": {{ 4242 | json }}
	},
	"db": {
		"driver": "sqlite",
		"options": {
			"path": {{ file "SECRET_FILE" | json }}
		}
	}
}
`

func TestLoadFromTemplateFile(t *testing.T) {
	secretFile, err := os.CreateTemp(t.TempDir(), "solve-test-secret-")
	if err != nil {
		t.Error("Error: ", err)
	}
	func() {
		defer func() { _ = secretFile.Close() }()
		if _, err := secretFile.Write([]byte("secret")); err != nil {
			t.Fatal("Error: ", err)
		}
	}()
	file, err := os.CreateTemp(t.TempDir(), "solve-test-")
	if err != nil {
		t.Error("Error: ", err)
	}
	func() {
		defer func() { _ = file.Close() }()
		if _, err := file.Write([]byte(strings.ReplaceAll(
			templateConfig, "SECRET_FILE", secretFile.Name(),
		))); err != nil {
			t.Fatal("Error: ", err)
		}
	}()
	cfg, err := LoadFromFile(file.Name())
	if err != nil {
		t.Fatal("Error: ", err)
	}
	if opts, ok := cfg.DB.Options.(SQLiteOptions); !ok {
		t.Fatalf("Invalid options type: %T", cfg.DB.Options)
	} else {
		testExpect(t, opts.Path, "secret")
	}
	testExpect(t, cfg.LogLevel, LogLevel(log.INFO))
}

func TestLoadFromInvalidFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "solve-test-")
	if err != nil {
		t.Error("Error: ", err)
	}
	func() {
		defer func() { _ = file.Close() }()
		if _, err := file.Write([]byte("invalid data")); err != nil {
			t.Fatal("Error: ", err)
		}
	}()
	if _, err := LoadFromFile(file.Name()); err == nil {
		t.Fatal("Expected error for invalid config file")
	}
}

func TestLoadFromInvalidTemplateFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "solve-test-")
	if err != nil {
		t.Error("Error: ", err)
	}
	func() {
		defer func() { _ = file.Close() }()
		if _, err := file.Write([]byte(`{"server": {{ invalid }} }`)); err != nil {
			t.Fatal("Error: ", err)
		}
	}()
	if _, err := LoadFromFile(file.Name()); err == nil {
		t.Fatal("Expected error for invalid config file")
	}
}

func TestLoadFromInvalidTemplateFile2(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "solve-test-")
	if err != nil {
		t.Error("Error: ", err)
	}
	func() {
		defer func() { _ = file.Close() }()
		_, err = file.Write([]byte(`{"server": { {{ .unknown }} } }`))
		if err != nil {
			t.Fatal("Error: ", err)
		}
	}()
	if _, err := LoadFromFile(file.Name()); err == nil {
		t.Fatal("Expected error for invalid config file")
	}
}

func TestServerAddress(t *testing.T) {
	s := Server{Host: "localhost", Port: 8080}
	testExpect(t, s.Address(), "localhost:8080")
}

func testExpect[T comparable](tb testing.TB, output, answer T) {
	if output != answer {
		tb.Fatalf(
			"Expected %q, got %q",
			fmt.Sprint(answer), fmt.Sprint(output),
		)
	}
}

package core

import (
	"testing"

	"github.com/udovin/solve/config"
)

func TestNewApp(t *testing.T) {
	var cfg config.Config
	if _, err := NewApp(&cfg); err == nil {
		t.Error("Expected error while creating app")
	}
	cfg.Database = config.DatabaseConfig{
		Driver: config.SQLiteDriver,
	}
	if _, err := NewApp(&cfg); err == nil {
		t.Error("Expected error while creating app")
	}
	cfg.Database = config.DatabaseConfig{
		Driver:  config.SQLiteDriver,
		Options: config.SQLiteOptions{Path: "?mode=memory"},
	}
	if _, err := NewApp(&cfg); err == nil {
		t.Error("Expected error while creating app")
	}
	cfg.Security = config.SecurityConfig{
		PasswordSalt: config.Secret{
			Type: config.DataSecret,
			Data: "qwerty123",
		},
	}
	if _, err := NewApp(&cfg); err != nil {
		t.Error("Error: ", err)
	}
}

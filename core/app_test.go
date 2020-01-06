package core

import (
	"testing"

	"github.com/udovin/solve/config"
)

func TestNewApp(t *testing.T) {
	var cfg config.Config
	if _, err := NewApp(cfg); err == nil {
		t.Fatal("Expected error while creating app")
	}
	cfg.DB = config.DB{
		Driver: config.SQLiteDriver,
	}
	if _, err := NewApp(cfg); err == nil {
		t.Fatal("Expected error while creating app")
	}
	cfg.DB = config.DB{
		Driver:  config.SQLiteDriver,
		Options: config.SQLiteOptions{Path: "?mode=memory"},
	}
	cfg.Security = config.Security{
		PasswordSalt: config.Secret{
			Type: config.DataSecret,
			Data: "qwerty123",
		},
	}
	if _, err := NewApp(cfg); err != nil {
		t.Fatal("Error:", err)
	}
}

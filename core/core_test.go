package core

import (
	"testing"

	"github.com/udovin/solve/config"
)

func TestNewCore(t *testing.T) {
	var cfg config.Config
	if _, err := NewCore(cfg); err == nil {
		t.Fatal("Expected error while creating core")
	}
	cfg.DB = config.DB{
		Driver: config.SQLiteDriver,
	}
	if _, err := NewCore(cfg); err == nil {
		t.Fatal("Expected error while creating core")
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
	if _, err := NewCore(cfg); err != nil {
		t.Fatal("Error:", err)
	}
}

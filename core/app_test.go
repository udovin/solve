package core

import (
	"testing"

	"github.com/udovin/solve/config"
)

func TestNewApp(t *testing.T) {
	var cfg config.Config
	if _, err := NewApp(&cfg); err == nil {
		t.Fatal("Expected error while creating app")
	}
	cfg.Database = config.DatabaseConfig{
		Driver: config.SQLiteDriver,
	}
	if _, err := NewApp(&cfg); err == nil {
		t.Fatal("Expected error while creating app")
	}
	cfg.Database = config.DatabaseConfig{
		Driver:  config.SQLiteDriver,
		Options: config.SQLiteOptions{Path: "?mode=memory"},
	}
	if _, err := NewApp(&cfg); err == nil {
		t.Fatal("Expected error while creating app")
	}
	cfg.Security = config.SecurityConfig{
		PasswordSalt: config.Secret{
			Type: config.DataSecret,
			Data: "qwerty123",
		},
	}
	if _, err := NewApp(&cfg); err != nil {
		t.Fatal("Error: ", err)
	}
}

func TestApp_StartStop(t *testing.T) {
	cfg := config.Config{
		Database: config.DatabaseConfig{
			Driver:  config.SQLiteDriver,
			Options: config.SQLiteOptions{Path: "?mode=memory"},
		},
		Security: config.SecurityConfig{
			PasswordSalt: config.Secret{
				Type: config.DataSecret,
				Data: "qwerty123",
			},
		},
	}
	app, err := NewApp(&cfg)
	if err != nil {
		t.Fatal("Error: ", err)
	}
	if err := app.Start(); err != nil {
		t.Fatal("Error: ", err)
	}
	app.Stop()
}

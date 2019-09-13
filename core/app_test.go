package core

import (
	"testing"
	"time"

	"github.com/udovin/solve/config"
)

func TestNewApp(t *testing.T) {
	var cfg config.Config
	if _, err := NewApp(&cfg); err == nil {
		t.Fatal("Expected error while creating app")
	}
	cfg.DB = config.DB{
		Driver: config.SQLiteDriver,
	}
	if _, err := NewApp(&cfg); err == nil {
		t.Fatal("Expected error while creating app")
	}
	cfg.DB = config.DB{
		Driver:  config.SQLiteDriver,
		Options: config.SQLiteOptions{Path: "?mode=memory"},
	}
	if _, err := NewApp(&cfg); err == nil {
		t.Fatal("Expected error while creating app")
	}
	cfg.Security = config.Security{
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
		DB: config.DB{
			Driver:  config.SQLiteDriver,
			Options: config.SQLiteOptions{Path: "?mode=memory"},
		},
		Security: config.Security{
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
	time.Sleep(2 * time.Second)
	app.Stop()
}

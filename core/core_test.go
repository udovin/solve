package core_test

import (
	"context"
	"testing"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/migrations"
)

func TestNewCore(t *testing.T) {
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
	c, err := core.NewCore(cfg)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := c.SetupAllManagers(); err != nil {
		t.Fatal("Error:", err)
	}
	if err := migrations.Apply(c); err != nil {
		t.Fatal("Error:", err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatal("Error:", err)
	}
	defer c.Stop()
}

func TestNewCore_Failure(t *testing.T) {
	var cfg config.Config
	if _, err := core.NewCore(cfg); err == nil {
		t.Fatal("Expected error while creating core")
	}
	cfg.DB = config.DB{
		Driver: config.SQLiteDriver,
	}
	if _, err := core.NewCore(cfg); err == nil {
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
	if _, err := core.NewCore(cfg); err != nil {
		t.Fatal("Error:", err)
	}
}

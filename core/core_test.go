package core_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/db"
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
	if err := c.Start(); err != nil {
		t.Fatal("Error:", err)
	}
	defer c.Stop()
	// Check that we can not start core twice.
	if err := c.Start(); err == nil {
		t.Fatal("Expected error")
	}
	// Check that we can stop core twice without no side effects.
	c.Stop()
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

func TestCore_WithTx(t *testing.T) {
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
	if err := c.Start(); err != nil {
		t.Fatal("Error:", err)
	}
	defer c.Stop()
	if err := c.WithTx(context.Background(), func(tx *sql.Tx) error {
		return fmt.Errorf("test error")
	}); err == nil {
		t.Fatal("Expected error")
	}
}

func TestGetDriver(t *testing.T) {
	if v := core.GetDialect(config.SQLiteDriver); v != db.SQLite {
		t.Fatalf("Expected %q, got %q", db.SQLite, v)
	}
	if v := core.GetDialect(config.PostgresDriver); v != db.Postgres {
		t.Fatalf("Expected %q, got %q", db.Postgres, v)
	}
}

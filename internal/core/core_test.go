package core

import (
	"context"
	"fmt"
	"testing"

	"github.com/udovin/solve/internal/config"
	"github.com/udovin/solve/internal/db"
	"github.com/udovin/solve/internal/migrations"
)

var testCfg = config.Config{
	DB: config.DB{
		Options: config.SQLiteOptions{Path: ":memory:"},
	},
	Security: &config.Security{
		PasswordSalt: "qwerty123",
	},
}

func TestNewCore(t *testing.T) {
	c, err := NewCore(testCfg)
	if err != nil {
		t.Fatal("Error:", err)
	}
	c.SetupAllStores()
	if err := db.ApplyMigrations(context.Background(), c.DB, "solve", migrations.Schema); err != nil {
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
	if _, err := NewCore(cfg); err == nil {
		t.Fatal("Expected error while creating core")
	}
	cfg.DB = config.DB{}
	if _, err := NewCore(cfg); err == nil {
		t.Fatal("Expected error while creating core")
	}
	cfg.DB = config.DB{
		Options: config.SQLiteOptions{Path: ":memory:"},
	}
	cfg.Security = &config.Security{
		PasswordSalt: "qwerty123",
	}
	if _, err := NewCore(cfg); err != nil {
		t.Fatal("Error:", err)
	}
}

func TestCore_WithTx(t *testing.T) {
	c, err := NewCore(testCfg)
	if err != nil {
		t.Fatal("Error:", err)
	}
	c.SetupAllStores()
	if err := db.ApplyMigrations(context.Background(), c.DB, "solve", migrations.Schema); err != nil {
		t.Fatal("Error:", err)
	}
	if err := c.Start(); err != nil {
		t.Fatal("Error:", err)
	}
	defer c.Stop()
	if err := c.WrapTx(context.Background(), func(context.Context) error {
		return fmt.Errorf("test error")
	}); err == nil {
		t.Fatal("Expected error")
	}
}

package core_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/migrations"
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
	c, err := core.NewCore(testCfg)
	if err != nil {
		t.Fatal("Error:", err)
	}
	c.SetupAllStores()
	manager, err := migrations.NewManager(c.DB)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := manager.Apply(context.Background()); err != nil {
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
	cfg.DB = config.DB{}
	if _, err := core.NewCore(cfg); err == nil {
		t.Fatal("Expected error while creating core")
	}
	cfg.DB = config.DB{
		Options: config.SQLiteOptions{Path: ":memory:"},
	}
	cfg.Security = &config.Security{
		PasswordSalt: "qwerty123",
	}
	if _, err := core.NewCore(cfg); err != nil {
		t.Fatal("Error:", err)
	}
}

func TestCore_WithTx(t *testing.T) {
	c, err := core.NewCore(testCfg)
	if err != nil {
		t.Fatal("Error:", err)
	}
	c.SetupAllStores()
	manager, err := migrations.NewManager(c.DB)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := manager.Apply(context.Background()); err != nil {
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

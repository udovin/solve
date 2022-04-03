package core_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/migrations"
	"github.com/udovin/solve/models"
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
	if err := migrations.Apply(c); err != nil {
		t.Fatal("Error:", err)
	}
	if err := c.Start(); err != nil {
		t.Fatal("Error:", err)
	}
	defer c.Stop()
	if err := c.WrapTx(context.Background(), func(context.Context) error {
		return fmt.Errorf("test error")
	}, nil); err == nil {
		t.Fatal("Expected error")
	}
}

func TestCore_Roles(t *testing.T) {
	c, err := core.NewCore(testCfg)
	if err != nil {
		t.Fatal("Error:", err)
	}
	c.SetupAllStores()
	if err := migrations.Apply(c); err != nil {
		t.Fatal("Error:", err)
	}
	if err := c.Start(); err != nil {
		t.Fatal("Error:", err)
	}
	defer c.Stop()
	roles, err := c.GetGuestRoles()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if ok, err := c.HasRole(roles, models.GuestGroupRole); err != nil {
		t.Fatal("Error:", err)
	} else if !ok {
		t.Fatalf("Role %q should exist", models.GuestGroupRole)
	}
}

func TestCore_Roles_NoRows(t *testing.T) {
	c, err := core.NewCore(testCfg)
	if err != nil {
		t.Fatal("Error:", err)
	}
	c.SetupAllStores()
	if err := migrations.Apply(c); err != nil {
		t.Fatal("Error:", err)
	}
	if err := c.Start(); err != nil {
		t.Fatal("Error:", err)
	}
	defer c.Stop()
	role, err := c.Roles.GetByName(models.GuestGroupRole)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := c.Roles.Delete(context.Background(), role.ID); err != nil {
		t.Fatal("Error:", err)
	}
	if err := c.Roles.Sync(context.Background()); err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := c.GetGuestRoles(); err != sql.ErrNoRows {
		t.Fatalf("Expected %q, got %q", sql.ErrNoRows, err)
	}
	if _, err := c.HasRole(core.RoleSet{}, "unknown"); err != sql.ErrNoRows {
		t.Fatalf("Expected %q, got %q", sql.ErrNoRows, err)
	}
	if _, err := c.GetAccountRoles(0); err != nil {
		t.Fatal("Error:", err)
	}
}

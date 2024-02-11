package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/udovin/solve/internal/config"
	"github.com/udovin/solve/internal/core"
	"github.com/udovin/solve/internal/db"
	"github.com/udovin/solve/internal/migrations"
)

var (
	testConfigFile *os.File
	testConfig     = config.Config{
		DB: config.DB{
			Options: config.SQLiteOptions{Path: ":memory:"},
		},
		Server:  &config.Server{},
		Invoker: &config.Invoker{},
		Security: &config.Security{
			PasswordSalt: "qwerty123",
		},
		Storage: &config.Storage{},
	}
)

func testSetup(tb testing.TB) {
	testConfig.SocketFile = filepath.Join(
		tb.TempDir(), "solve-server.sock",
	)
	testConfig.Storage.Options = config.LocalStorageOptions{
		FilesDir: filepath.Join(tb.TempDir(), "files"),
	}
	var err error
	func() {
		testConfigFile, err = os.CreateTemp(tb.TempDir(), "test-")
		if err != nil {
			tb.Fatal("Error:", err)
		}
		defer testConfigFile.Close()
		err := json.NewEncoder(testConfigFile).Encode(testConfig)
		if err != nil {
			tb.Fatal("Error:", err)
		}
	}()
	c, err := core.NewCore(testConfig)
	if err != nil {
		tb.Fatal("Error:", err)
	}
	c.SetupAllStores()
	if err := db.ApplyMigrations(
		context.Background(), c.DB, "solve", migrations.Schema,
	); err != nil {
		tb.Fatal("Error:", err)
	}
}

func testTeardown(tb testing.TB) {
	c, err := core.NewCore(testConfig)
	if err != nil {
		tb.Fatal("Error:", err)
	}
	c.SetupAllStores()
	if err := db.ApplyMigrations(
		context.Background(), c.DB, "solve", migrations.Schema,
		db.WithZeroMigration,
	); err != nil {
		tb.Fatal("Error:", err)
	}
}

func TestServerMain(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	cmd := cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().Set("config", testConfigFile.Name())
	go testCancel()
	serverMain(&cmd, nil)
}

func TestMigrateMain(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	cmd := cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().String("from", "", "")
	cmd.Flags().Set("config", testConfigFile.Name())
	go testCancel()
	migrateMain(&cmd, nil)
}

func TestMigrateDataMain(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	cmd := cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().String("from", "", "")
	cmd.Flags().Set("config", testConfigFile.Name())
	go testCancel()
	migrateDataMain(&cmd, nil)
}

func TestVersionMain(t *testing.T) {
	cmd := cobra.Command{}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Unexpected panic: %v", r)
		}
	}()
	versionMain(&cmd, nil)
}

func TestGetConfigUnknown(t *testing.T) {
	cmd := cobra.Command{}
	if _, err := getConfig(&cmd); err == nil {
		t.Fatal("Expected error")
	}
}

func TestCommand(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic")
		}
	}()
	args := os.Args
	os.Args = []string{"solve", "--config", "not-found", "server"}
	main()
	os.Args = args
}

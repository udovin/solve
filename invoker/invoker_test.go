package invoker

import (
	"context"
	"testing"
	"time"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/migrations"
)

var testInvoker *Invoker

func testSetup(tb testing.TB) {
	cfg := config.Config{
		DB: config.DB{
			Options: config.SQLiteOptions{Path: ":memory:"},
		},
		Invoker: &config.Invoker{},
		Security: &config.Security{
			PasswordSalt: "qwerty123",
		},
		Storage: &config.Storage{
			FilesDir: tb.TempDir(),
		},
	}
	c, err := core.NewCore(cfg)
	if err != nil {
		tb.Fatal("Error:", err)
	}
	c.SetupAllStores()
	if err := db.ApplyMigrations(context.Background(), c.DB, "solve", migrations.Schema); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := c.Start(); err != nil {
		tb.Fatal("Error:", err)
	}
	testInvoker = New(c)
}

func testTeardown(tb testing.TB) {
	_ = db.ApplyMigrations(context.Background(), testInvoker.core.DB, "solve", migrations.Schema, db.WithZeroMigration)
	testInvoker.core.Stop()
}

func TestInvoker_Start(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	testInvoker.Start()
	// Wait for cache sync.
	<-time.After(1100 * time.Millisecond)
}

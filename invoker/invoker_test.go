package invoker

import (
	"testing"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/migrations"
)

var testInvoker *Invoker

func testSetup(tb testing.TB) {
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
		tb.Fatal("Error:", err)
	}
	if err := c.SetupAllStores(); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := migrations.Apply(c); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := c.Start(); err != nil {
		tb.Fatal("Error:", err)
	}
	testInvoker = New(c)
}

func testTeardown(tb testing.TB) {
	testInvoker.core.Stop()
}

func TestInvoker_Start(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	testInvoker.Start()
}

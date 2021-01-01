package migrations_test

import (
	"testing"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/migrations"
)

var testCfg = config.Config{
	DB: config.DB{
		Driver:  config.SQLiteDriver,
		Options: config.SQLiteOptions{Path: ":memory:"},
	},
	Security: config.Security{
		PasswordSalt: config.Secret{
			Type: config.DataSecret,
			Data: "qwerty123",
		},
	},
}

func TestMigrations(t *testing.T) {
	c, err := core.NewCore(testCfg)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := c.SetupAllStores(); err != nil {
		t.Fatal("Error:", err)
	}
	if err := migrations.Apply(c); err != nil {
		t.Fatal("Error:", err)
	}
	if err := migrations.Unapply(c); err != nil {
		t.Fatal("Error:", err)
	}
}

package migrations

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/udovin/solve/config"
)

func TestMigrations(t *testing.T) {
	cfg := config.Config{
		DB: config.DB{
			Options: config.SQLiteOptions{Path: ":memory:"},
		},
		Security: &config.Security{
			PasswordSalt: "qwerty123",
		},
	}
	db, err := cfg.DB.Create()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := Apply(context.Background(), db); err != nil {
		t.Fatal("Error:", err)
	}
	if err := Apply(context.Background(), db, WithZero); err != nil {
		t.Fatal("Error:", err)
	}
}

func TestPostgresMigrations(t *testing.T) {
	pgHost, ok := os.LookupEnv("POSTGRES_HOST")
	if !ok {
		t.Skip()
	}
	pgPortStr, ok := os.LookupEnv("POSTGRES_PORT")
	if !ok {
		t.Skip()
	}
	pgPort, err := strconv.Atoi(pgPortStr)
	if err != nil {
		t.Fatal("Error:", err)
	}
	cfg := config.Config{
		DB: config.DB{
			Options: config.PostgresOptions{
				Host:     pgHost,
				Port:     pgPort,
				User:     "postgres",
				Password: "postgres",
				Name:     "postgres",
				SSLMode:  "disable",
			},
		},
		Security: &config.Security{
			PasswordSalt: "qwerty123",
		},
	}
	db, err := cfg.DB.Create()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := Apply(context.Background(), db); err != nil {
		t.Fatal("Error:", err)
	}
	if err := Apply(context.Background(), db, WithZero); err != nil {
		t.Fatal("Error:", err)
	}
}

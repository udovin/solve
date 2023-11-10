package db_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/udovin/solve/internal/config"
	"github.com/udovin/solve/internal/db"
	"github.com/udovin/solve/internal/migrations"
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
	conn, err := cfg.DB.Create()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := db.ApplyMigrations(context.Background(), conn, "solve", migrations.Schema); err != nil {
		t.Fatal("Error:", err)
	}
	if err := db.ApplyMigrations(context.Background(), conn, "solve", migrations.Schema, db.WithZeroMigration); err != nil {
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
				Hosts:    []string{fmt.Sprintf("%s:%d", pgHost, pgPort)},
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
	conn, err := cfg.DB.Create()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := db.ApplyMigrations(context.Background(), conn, "solve", migrations.Schema); err != nil {
		t.Fatal("Error:", err)
	}
	if err := db.ApplyMigrations(context.Background(), conn, "solve", migrations.Schema, db.WithZeroMigration); err != nil {
		t.Fatal("Error:", err)
	}
}

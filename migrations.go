package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/db"
)

type dbMigration struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

func (o dbMigration) ObjectID() int64 {
	return o.ID
}

func applyDBMigration(conn *sql.DB, query string) error {
	if _, err := conn.Exec(query); err != nil {
		return err
	}
	return nil
}

// setupDBMigrations creates migrations table if it does not exists.
func setupDBMigrations(conn *sql.DB, dialect db.Dialect) error {
	switch dialect {
	case db.SQLite:
		_, err := conn.Exec(
			`CREATE TABLE IF NOT EXISTS "solve_db_migration" (` +
				`"id" INTEGER PRIMARY KEY, "name" VARCHAR(255) NOT NULL)`,
		)
		return err
	case db.Postgres:
		_, err := conn.Exec(
			`CREATE TABLE IF NOT EXISTS "solve_db_migration" (` +
				`"id" SERIAL NOT NULL ` +
				`CONSTRAINT solve_db_migration_pkey PRIMARY KEY, ` +
				`"name" VARCHAR(255) NOT NULL)`,
		)
		return err
	default:
		return fmt.Errorf("unsupported dialect %q", dialect)
	}
}

// applyDBMigrations applies all migrations to the database.
func applyDBMigrations(
	conn *sql.DB, dialect db.Dialect, migrations db.ObjectStore,
) error {
	// List migrations.
	files, err := ioutil.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, file := range files {
		switch dialect {
		case db.SQLite:
			if !strings.HasSuffix(file.Name(), "_sqlite.sql") {
				continue
			}
		case db.Postgres:
			if !strings.HasSuffix(file.Name(), "_postgres.sql") {
				continue
			}
		}
		query, err := ioutil.ReadFile(filepath.Join("migrations", file.Name()))
		if err != nil {
			panic(err)
		}
		// Check that migration already applied.
		tx, err := conn.Begin()
		if err != nil {
			panic(err)
		}
		rows, err := migrations.FindObjects(tx, `"name" = $1`, file.Name())
		if err != nil {
			_ = tx.Rollback()
			panic(err)
		}
		if rows.Next() {
			_ = rows.Close()
			_ = tx.Rollback()
			continue
		}
		_ = rows.Close()
		_ = tx.Rollback()
		// Apply migration.
		if err := applyDBMigration(conn, string(query)); err != nil {
			panic(err)
		}
		// Save migration to database.
		tx, err = conn.Begin()
		if err != nil {
			panic(err)
		}
		_, err = migrations.CreateObject(tx, dbMigration{Name: file.Name()})
		if err != nil {
			_ = tx.Rollback()
		}
		if err := tx.Commit(); err != nil {
			panic(err)
		}
	}
	return nil
}

func upgradeDbMain(cmd *cobra.Command, _ []string) {
	// Load config from file.
	cfg, err := getConfig(cmd)
	if err != nil {
		panic(err)
	}
	// Setup database connection.
	conn, err := cfg.DB.Create()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = conn.Close()
	}()
	// Detect database dialect.
	dialect := core.GetDialect(cfg.DB.Driver)
	// Setup migrations table.
	if err := setupDBMigrations(conn, dialect); err != nil {
		panic(err)
	}
	// Setup migrations store.
	migrations := db.NewObjectStore(
		dbMigration{}, "id", "solve_db_migration", dialect,
	)
	// Apply all migrations.
	if err := applyDBMigrations(conn, dialect, migrations); err != nil {
		panic(err)
	}
}

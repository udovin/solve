package config

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/udovin/solve/tools"

	// Register SQL drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type DatabaseDriver string

const (
	SQLiteDriver   DatabaseDriver = "SQLite"
	PostgresDriver DatabaseDriver = "Postgres"
)

// Configuration for database connection
type DatabaseConfig struct {
	Driver  DatabaseDriver `json:""`
	Options interface{}    `json:""`
}

// SQLite connection options
type SQLiteOptions struct {
	Path string `json:""`
}

// Postgres connection options
type PostgresOptions struct {
	Host     string `json:""`
	Port     int    `json:""`
	User     string `json:""`
	Password Secret `json:""`
	Name     string `json:""`
}

// Parse JSON to create appropriate connection configuration
func (c *DatabaseConfig) UnmarshalJSON(bytes []byte) error {
	var g struct {
		Driver  DatabaseDriver           `json:""`
		Options tools.InterfaceUnmarshal `json:""`
	}
	if err := json.Unmarshal(bytes, &g); err != nil {
		return err
	}
	switch g.Driver {
	case SQLiteDriver:
		var options SQLiteOptions
		if err := json.Unmarshal(g.Options, &options); err != nil {
			return err
		}
		c.Options = options
	case PostgresDriver:
		var options PostgresOptions
		if err := json.Unmarshal(g.Options, &options); err != nil {
			return err
		}
		c.Options = options
	default:
		return fmt.Errorf("driver '%s' is not supported", g.Driver)
	}
	c.Driver = g.Driver
	return nil
}

func createSQLiteDB(opts SQLiteOptions) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s", opts.Path))
	if err != nil {
		return nil, err
	}
	// This can increase writes performance
	_, err = db.Exec(`PRAGMA journal_mode=WAL`)
	if err != nil {
		// Dont forget to close connection on failure
		_ = db.Close()
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func createPostgresDB(opts PostgresOptions) (*sql.DB, error) {
	password, err := opts.Password.GetValue()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s",
		opts.Host, opts.Port, opts.User, password, opts.Name,
	))
	return db, err
}

// Creates database connection using current configuration
func (c *DatabaseConfig) CreateDB() (*sql.DB, error) {
	switch t := c.Options.(type) {
	case SQLiteOptions:
		return createSQLiteDB(t)
	case PostgresOptions:
		return createPostgresDB(t)
	default:
		return nil, errors.New("unsupported database config type")
	}
}

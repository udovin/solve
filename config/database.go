package config

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	// Register SQL drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// DBDriver represents database driver name.
type DBDriver string

const (
	// SQLiteDriver represents SQLite driver.
	SQLiteDriver DBDriver = "sqlite"
	// PostgresDriver represents Postgres driver.
	PostgresDriver DBDriver = "postgres"
)

// DB stores configuration for database connection.
type DB struct {
	// Driver contains database driver name.
	Driver DBDriver `json:"driver"`
	// Options contains options for database driver.
	//
	// For SQLiteDriver field should contains SQLiteOptions.
	// For PostgresDriver field should contains PostgresOptions.
	Options interface{} `json:"options"`
}

// SQLiteOptions stores SQLite connection options.
type SQLiteOptions struct {
	// Path contains path to SQLite database file.
	Path string `json:"path"`
}

// PostgresOptions stores Postgres connection options.
type PostgresOptions struct {
	// Host contains host address.
	Host string `json:"host"`
	// Port contains port address.
	Port int `json:"port"`
	// User contains username of user.
	User string `json:"user"`
	// Password contains password of user.
	Password Secret `json:"password"`
	// Name contains name of database.
	Name string `json:"name"`
}

// UnmarshalJSON parses JSON to create appropriate connection configuration.
func (c *DB) UnmarshalJSON(bytes []byte) error {
	var g struct {
		Driver DBDriver `json:"driver"`
		// Options will be parsed after detecting driver name.
		Options json.RawMessage `json:"options"`
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
	return fixCreateSQLiteDB(
		sql.Open("sqlite3", fmt.Sprintf("file:%s", opts.Path)),
	)
}

func fixCreateSQLiteDB(db *sql.DB, err error) (*sql.DB, error) {
	if err != nil {
		return nil, err
	}
	// This can increase writes performance.
	if _, err = db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		// Dont forget to close connection on failure.
		_ = db.Close()
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func createPostgresDB(opts PostgresOptions) (*sql.DB, error) {
	password, err := opts.Password.Secret()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
		opts.Host, opts.Port, opts.User, password, opts.Name,
	))
	return db, err
}

// Create creates database connection using current configuration.
func (c *DB) Create() (*sql.DB, error) {
	switch t := c.Options.(type) {
	case SQLiteOptions:
		return createSQLiteDB(t)
	case PostgresOptions:
		return createPostgresDB(t)
	default:
		return nil, errors.New("unsupported database config type")
	}
}

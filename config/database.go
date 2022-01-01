package config

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/udovin/gosql"

	// Register SQL drivers
	_ "github.com/jackc/pgx/v4/stdlib"
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
	Password string `json:"password"`
	// Name contains name of database.
	Name string `json:"name"`
	// SSLMode contains sslmode configuration.
	SSLMode string `json:"sslmode"`
}

// DB stores configuration for database connection.
type DB struct {
	// Options contains options for database driver.
	//
	// For SQLiteDriver field should contains SQLiteOptions.
	// For PostgresDriver field should contains PostgresOptions.
	Options any
}

// UnmarshalJSON parses JSON to create appropriate connection configuration.
func (c *DB) UnmarshalJSON(bytes []byte) error {
	var cfg struct {
		Driver  DBDriver        `json:"driver"`
		Options json.RawMessage `json:"options"`
	}
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return err
	}
	switch cfg.Driver {
	case SQLiteDriver:
		var options SQLiteOptions
		if err := json.Unmarshal(cfg.Options, &options); err != nil {
			return err
		}
		c.Options = options
	case PostgresDriver:
		var options PostgresOptions
		if err := json.Unmarshal(cfg.Options, &options); err != nil {
			return err
		}
		c.Options = options
	default:
		return fmt.Errorf("driver %q is not supported", cfg.Driver)
	}
	return nil
}

func (c DB) MarshalJSON() ([]byte, error) {
	cfg := struct {
		Driver  DBDriver `json:"driver"`
		Options any      `json:"options"`
	}{
		Options: c.Options,
	}
	switch t := c.Options.(type) {
	case SQLiteOptions:
		cfg.Driver = SQLiteDriver
	case PostgresOptions:
		cfg.Driver = PostgresDriver
	default:
		return nil, fmt.Errorf("options of type %T is not supported", t)
	}
	return json.Marshal(cfg)
}

func createSQLiteDB(opts SQLiteOptions) (*gosql.DB, error) {
	return (gosql.SQLiteConfig{
		Path: opts.Path,
	}).NewDB()
}

func createPostgresDB(opts PostgresOptions) (*gosql.DB, error) {
	return (gosql.PostgresConfig{
		Hosts:    []string{fmt.Sprintf("%s:%d", opts.Host, opts.Port)},
		User:     opts.User,
		Password: opts.Password,
		Name:     opts.Name,
		SSLMode:  opts.SSLMode,
	}).NewDB()
}

// Create creates database connection using current configuration.
func (c *DB) Create() (*gosql.DB, error) {
	switch v := c.Options.(type) {
	case SQLiteOptions:
		return createSQLiteDB(v)
	case PostgresOptions:
		return createPostgresDB(v)
	default:
		return nil, errors.New("unsupported database config type")
	}
}

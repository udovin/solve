package migrations

import (
	"database/sql"
	"fmt"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/db"
)

// Migration represents database migration.
type Migration interface {
	// Name should return unique migration name.
	Name() string
	// Apply should apply database migration.
	Apply(c *core.Core, tx *sql.Tx) error
	// Unapply should unapply database migration.
	Unapply(c *core.Core, tx *sql.Tx) error
}

// migrations contains list of all migrations.
var migrations = []Migration{
	&m001{},
}

type migration struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

func (o migration) ObjectID() int64 {
	return o.ID
}

const migrationTable = "solve_db_migration"

// Apply applies all migrations to the specified core.
func Apply(c *core.Core) error {
	dialect := core.GetDialect(c.Config.DB.Driver)
	// Prepare database.
	if err := c.WithTx(func(tx *sql.Tx) error {
		return setupDB(tx, dialect)
	}); err != nil {
		return err
	}
	// Prepare migration store.
	store := db.NewObjectStore(migration{}, "id", migrationTable, dialect)
	for _, m := range migrations {
		if err := c.WithTx(func(tx *sql.Tx) error {
			// Check that migration already applied.
			if applied, err := isApplied(store, tx, m.Name()); err != nil {
				return err
			} else if applied {
				return nil
			}
			// Apply migration.
			if err := m.Apply(c, tx); err != nil {
				return err
			}
			// Save to database that migration was applied.
			_, err := store.CreateObject(tx, migration{Name: m.Name()})
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

// Unapply rollbacks all applied migrations for specified core.
func Unapply(c *core.Core) error {
	dialect := core.GetDialect(c.Config.DB.Driver)
	// Prepare database.
	if err := c.WithTx(func(tx *sql.Tx) error {
		return setupDB(tx, dialect)
	}); err != nil {
		return err
	}
	// Prepare migration store.
	store := db.NewObjectStore(migration{}, "id", migrationTable, dialect)
	for i := len(migrations) - 1; i >= 0; i-- {
		m := migrations[i]
		if err := c.WithTx(func(tx *sql.Tx) error {
			// Check that migration already applied.
			if applied, err := isApplied(store, tx, m.Name()); err != nil {
				return err
			} else if !applied {
				return nil
			}
			// Apply migration.
			if err := m.Unapply(c, tx); err != nil {
				return err
			}
			// Remove migration from database.
			var ids []int64
			if err := func() error {
				rows, err := store.FindObjects(tx, `"name" = $1`, m.Name())
				if err != nil {
					return err
				}
				defer func() { _ = rows.Close() }()
				for rows.Next() {
					ids = append(ids, rows.Object().ObjectID())
				}
				return rows.Err()
			}(); err != nil {
				return err
			}
			for _, id := range ids {
				if err := store.DeleteObject(tx, id); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func isApplied(s db.ObjectROStore, tx *sql.Tx, name string) (bool, error) {
	rows, err := s.FindObjects(tx, `"name" = $1`, name)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = rows.Close()
	}()
	return rows.Next(), nil
}

// setupDB creates migrations table if it does not exists.
func setupDB(tx *sql.Tx, dialect db.Dialect) error {
	switch dialect {
	case db.SQLite:
		_, err := tx.Exec(fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %q (`+
				`"id" INTEGER PRIMARY KEY, "name" VARCHAR(255) NOT NULL)`,
			migrationTable,
		))
		return err
	case db.Postgres:
		_, err := tx.Exec(fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %q (`+
				`"id" SERIAL NOT NULL `+
				`CONSTRAINT solve_db_migration_pkey PRIMARY KEY, `+
				`"name" VARCHAR(255) NOT NULL)`,
			migrationTable,
		))
		return err
	default:
		return fmt.Errorf("unsupported dialect %q", dialect)
	}
}

// Package migrations contains migrations for solve database.
package migrations

import (
	"context"
	"time"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/db/schema"
)

// Migration represents database migration.
type Migration interface {
	// Name should return unique migration name.
	Name() string
	// Apply should apply database migration.
	Apply(ctx context.Context, c *core.Core) error
	// Unapply should unapply database migration.
	Unapply(ctx context.Context, c *core.Core) error
}

// migrations contains list of all migrations.
var migrations = []Migration{
	&m001{},
}

type migration struct {
	ID      int64  `db:"id"`
	Name    string `db:"name"`
	Version string `db:"version"`
	Time    int64  `db:"time"`
}

func (o migration) ObjectID() int64 {
	return o.ID
}

const migrationTableName = "solve_db_migration"

// Apply applies all migrations to the specified core.
func Apply(c *core.Core) error {
	if err := setupMigrations(c.DB); err != nil {
		return err
	}
	store := db.NewObjectStore[migration]("id", migrationTableName, c.DB)
	for _, m := range migrations {
		if err := c.WrapTx(context.Background(), func(ctx context.Context) error {
			// Check that migration already applied.
			if applied, err := isApplied(ctx, store, m.Name()); err != nil {
				return err
			} else if applied {
				return nil
			}
			// Apply migration.
			if err := m.Apply(ctx, c); err != nil {
				return err
			}
			// Save to database that migration was applied.
			object := migration{
				Name:    m.Name(),
				Version: core.Version,
				Time:    time.Now().Unix(),
			}
			return store.CreateObject(ctx, &object)
		}); err != nil {
			return err
		}
	}
	return nil
}

// Unapply rollbacks last applied migration for specified core.
func Unapply(c *core.Core, all bool) error {
	if err := setupMigrations(c.DB); err != nil {
		return err
	}
	store := db.NewObjectStore[migration](
		"id", migrationTableName, c.DB,
	)
	stop := false
	for i := len(migrations) - 1; i >= 0 && !stop; i-- {
		m := migrations[i]
		if err := c.WrapTx(context.Background(), func(ctx context.Context) error {
			// Check that migration already applied.
			if applied, err := isApplied(ctx, store, m.Name()); err != nil {
				return err
			} else if !applied {
				return nil
			}
			// Apply migration.
			if err := m.Unapply(ctx, c); err != nil {
				return err
			}
			// Remove migration from database.
			var ids []int64
			if err := func() error {
				rows, err := store.FindObjects(
					ctx, gosql.Column("name").Equal(m.Name()),
				)
				if err != nil {
					return err
				}
				defer func() { _ = rows.Close() }()
				for rows.Next() {
					ids = append(ids, rows.Row().ObjectID())
				}
				return rows.Err()
			}(); err != nil {
				return err
			}
			for _, id := range ids {
				if err := store.DeleteObject(ctx, id); err != nil {
					return err
				}
			}
			stop = !all
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func isApplied(
	ctx context.Context, store db.ObjectROStore[migration], name string,
) (bool, error) {
	rows, err := store.FindObjects(ctx, gosql.Column("name").Equal(name))
	if err != nil {
		return false, err
	}
	defer func() {
		_ = rows.Close()
	}()
	return rows.Next(), nil
}

var mirgationTable = schema.Table{
	Name: migrationTableName,
	Columns: []schema.Column{
		{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
		{Name: "name", Type: schema.String},
		{Name: "version", Type: schema.String},
		{Name: "time", Type: schema.Int64},
	},
}

// setupMigrations creates migrations table if it does not exists.
func setupMigrations(db *gosql.DB) error {
	query, err := mirgationTable.BuildCreateSQL(db.Dialect(), false)
	if err != nil {
		return err
	}
	_, err = db.Exec(query)
	return err
}

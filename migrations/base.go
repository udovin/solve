// Package migrations contains migrations for solve database.
package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/db/schema"
)

// migrationImpl represents database migration.
type migrationImpl interface {
	// Name should return unique migration name.
	Name() string
	// Apply should apply database migration.
	Apply(ctx context.Context, conn *gosql.DB) error
	// Unapply should unapply database migration.
	Unapply(ctx context.Context, conn *gosql.DB) error
}

var registeredMigrations = map[string]migrationImpl{}

func registerMigration(m migrationImpl) {
	name := m.Name()
	if _, ok := registeredMigrations[name]; ok {
		panic(fmt.Errorf("migration %q already registered", name))
	}
	registeredMigrations[name] = m
}

type migrationState struct {
	Name      string
	Applied   bool
	Supported bool
}

type Manager struct {
	db    *gosql.DB
	store db.ObjectStore[migration, *migration]
}

func (m *Manager) init() error {
	query, err := mirgationTable.BuildCreateSQL(m.db.Dialect(), false)
	if err != nil {
		return err
	}
	_, err = m.db.Exec(query)
	return err
}

func (m *Manager) getMigrations() []migrationImpl {
	var migrations []migrationImpl
	for _, migration := range registeredMigrations {
		migrations = append(migrations, migration)
	}
	sort.Sort(migrationImplSorter(migrations))
	return migrations
}

func (m *Manager) getAppliedMigrations(ctx context.Context) ([]migration, error) {
	var migrations []migration
	rows, err := m.store.LoadObjects(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		migrations = append(migrations, rows.Row())
	}
	sort.Sort(migrationSorter(migrations))
	return migrations, rows.Err()
}

func (m *Manager) getState(ctx context.Context) ([]migrationState, error) {
	migrations := m.getMigrations()
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}
	var result []migrationState
	it, jt := 0, 0
	for it < len(migrations) && jt < len(applied) {
		if migrations[it].Name() < applied[jt].Name {
			result = append(result, migrationState{
				Name:      migrations[it].Name(),
				Applied:   false,
				Supported: true,
			})
			it++
		} else if applied[jt].Name < migrations[it].Name() {
			result = append(result, migrationState{
				Name:      applied[jt].Name,
				Applied:   true,
				Supported: false,
			})
			jt++
		} else {
			result = append(result, migrationState{
				Name:      applied[jt].Name,
				Applied:   true,
				Supported: true,
			})
			it++
			jt++
		}
	}
	for it < len(migrations) {
		result = append(result, migrationState{
			Name:      migrations[it].Name(),
			Applied:   false,
			Supported: true,
		})
		it++
	}
	for jt < len(applied) {
		result = append(result, migrationState{
			Name:      applied[jt].Name,
			Applied:   true,
			Supported: false,
		})
		jt++
	}
	return result, nil
}

type Option func(state []migrationState, endPos *int) error

func WithMigration(name string) Option {
	if name == "zero" {
		return WithZero
	}
	return func(state []migrationState, endPos *int) error {
		for i := 0; i < len(state); i++ {
			if state[i].Name == name {
				*endPos = i + 1
				return nil
			}
		}
		return fmt.Errorf("invalid migration %q", name)
	}
}

func WithZero(state []migrationState, endPos *int) error {
	*endPos = 0
	return nil
}

func (m *Manager) Apply(ctx context.Context, options ...Option) error {
	state, err := m.getState(ctx)
	if err != nil {
		return err
	}
	beginPos := 0
	for i := 0; i < len(state); i++ {
		if state[i].Applied {
			beginPos = i + 1
		}
	}
	endPos := len(state)
	for _, option := range options {
		if err := option(state, &endPos); err != nil {
			return err
		}
	}
	if endPos < beginPos {
		return m.applyBackward(ctx, state[endPos:beginPos])
	}
	return m.applyForward(ctx, state[beginPos:endPos])
}

func (m *Manager) applyForward(ctx context.Context, migrations []migrationState) error {
	if len(migrations) == 0 {
		fmt.Println("No migrations to apply")
		return nil
	}
	for _, mgr := range migrations {
		fmt.Println("Applying migration:", mgr.Name)
		impl, ok := registeredMigrations[mgr.Name]
		if !ok {
			return fmt.Errorf("migration %q is not supported", mgr.Name)
		}
		if err := gosql.WrapTx(ctx, m.db.DB, func(tx *sql.Tx) error {
			ctx := db.WithTx(ctx, tx)
			// Apply migration.
			if err := impl.Apply(ctx, m.db); err != nil {
				return err
			}
			// Save to database that migration was applied.
			object := migration{
				Name:    mgr.Name,
				Version: core.Version,
				Time:    time.Now().Unix(),
			}
			return m.store.CreateObject(ctx, &object)
		}); err != nil {
			return err
		}
		fmt.Println("Migration applied:", mgr.Name)
	}
	return nil
}

func (m *Manager) getAppliedMigration(ctx context.Context, name string) (migration, error) {
	rows, err := m.store.FindObjects(ctx, gosql.Column("name").Equal(name))
	if err != nil {
		return migration{}, err
	}
	defer func() {
		_ = rows.Close()
	}()
	if rows.Next() {
		return rows.Row(), nil
	}
	if err := rows.Err(); err != nil {
		return migration{}, err
	}
	return migration{}, sql.ErrNoRows
}

func (m *Manager) applyBackward(ctx context.Context, migrations []migrationState) error {
	if len(migrations) == 0 {
		fmt.Println("No migrations to reverse apply")
		return nil
	}
	for i := 0; i < len(migrations); i++ {
		mgr := migrations[len(migrations)-i-1]
		fmt.Println("Reverse applying migration:", mgr.Name)
		impl, ok := registeredMigrations[mgr.Name]
		if !ok {
			return fmt.Errorf("migration %q is not supported", mgr.Name)
		}
		if !mgr.Applied {
			return fmt.Errorf("migration %q is not applied", mgr.Name)
		}
		if err := gosql.WrapTx(ctx, m.db.DB, func(tx *sql.Tx) error {
			ctx := db.WithTx(ctx, tx)
			object, err := m.getAppliedMigration(ctx, mgr.Name)
			if err != nil {
				return err
			}
			if err := impl.Unapply(ctx, m.db); err != nil {
				return err
			}
			return m.store.DeleteObject(ctx, object.ID)
		}); err != nil {
			return err
		}
		fmt.Println("Migration reverse applied:", mgr.Name)
	}
	return nil
}

func NewManager(conn *gosql.DB) (*Manager, error) {
	m := &Manager{
		db:    conn,
		store: db.NewObjectStore[migration]("id", migrationTableName, conn),
	}
	err := m.init()
	return m, err
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

func (o *migration) SetObjectID(id int64) {
	o.ID = id
}

const migrationTableName = "solve_db_migration"

var mirgationTable = schema.Table{
	Name: migrationTableName,
	Columns: []schema.Column{
		{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
		{Name: "name", Type: schema.String},
		{Name: "version", Type: schema.String},
		{Name: "time", Type: schema.Int64},
	},
}

type migrationSorter []migration

func (v migrationSorter) Len() int {
	return len(v)
}

func (v migrationSorter) Less(i, j int) bool {
	return v[i].Name < v[j].Name
}

func (v migrationSorter) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

type migrationImplSorter []migrationImpl

func (v migrationImplSorter) Len() int {
	return len(v)
}

func (v migrationImplSorter) Less(i, j int) bool {
	return v[i].Name() < v[j].Name()
}

func (v migrationImplSorter) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

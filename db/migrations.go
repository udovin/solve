package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/udovin/gosql"
	"github.com/udovin/solve/config"
	"github.com/udovin/solve/db/schema"
)

// Migration represents database migration.
type Migration interface {
	// Apply should apply database migration.
	Apply(ctx context.Context, conn *gosql.DB) error
	// Unapply should unapply database migration.
	Unapply(ctx context.Context, conn *gosql.DB) error
}

type NamedMigration struct {
	Name      string
	Migration Migration
}

// MigrationGroup represents group of database migrations.
type MigrationGroup interface {
	// Register registers new migration to group.
	AddMigration(name string, m Migration)
	// GetMigration returns migration by name.
	GetMigration(name string) Migration
	// GetMigrations returns migrations by their order.
	GetMigrations() []NamedMigration
}

func NewMigration(operations []schema.Operation) Migration {
	return &simpleMigration{
		operations: operations,
	}
}

type simpleMigration struct {
	operations []schema.Operation
}

func (m *simpleMigration) Apply(ctx context.Context, conn *gosql.DB) error {
	tx := GetRunner(ctx, conn)
	for _, table := range m.operations {
		query, err := table.BuildApply(conn.Dialect())
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func (m *simpleMigration) Unapply(ctx context.Context, conn *gosql.DB) error {
	tx := GetRunner(ctx, conn)
	for i := 0; i < len(m.operations); i++ {
		table := m.operations[len(m.operations)-i-1]
		query, err := table.BuildUnapply(conn.Dialect())
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func NewMigrationGroup() MigrationGroup {
	return &migrationGroup{
		migrations: map[string]Migration{},
	}
}

type migrationGroup struct {
	migrations map[string]Migration
}

func (g *migrationGroup) AddMigration(name string, m Migration) {
	if _, ok := g.migrations[name]; ok {
		panic(fmt.Errorf("migration %q already exists", name))
	}
	g.migrations[name] = m
}

func (g *migrationGroup) GetMigration(name string) Migration {
	migration, ok := g.migrations[name]
	if !ok {
		panic(fmt.Errorf("migration %q does not exists", name))
	}
	return migration
}

func (g *migrationGroup) GetMigrations() []NamedMigration {
	var names []string
	for name := range g.migrations {
		names = append(names, name)
	}
	sort.Strings(names)
	var migrations []NamedMigration
	for _, name := range names {
		migration, ok := g.migrations[name]
		if !ok {
			panic(fmt.Errorf("migration %q does not exists", name))
		}
		migrations = append(migrations, NamedMigration{
			Name:      name,
			Migration: migration,
		})
	}
	return migrations
}

func ApplyMigrations(ctx context.Context, conn *gosql.DB, name string, g MigrationGroup, options ...MigrateOption) error {
	m := &manager{
		db:    conn,
		group: name,
		store: NewObjectStore[migration]("id", migrationTableName, conn),
	}
	if err := m.init(); err != nil {
		return err
	}
	return m.Apply(ctx, g, options...)
}

type migrationState struct {
	Name      string
	Applied   bool
	Supported bool
}

type manager struct {
	db    *gosql.DB
	group string
	store ObjectStore[migration, *migration]
}

func (m *manager) init() error {
	query, err := mirgationTable.BuildApply(m.db.Dialect())
	if err != nil {
		return err
	}
	_, err = m.db.Exec(query)
	return err
}

func (m *manager) getAppliedMigrations(ctx context.Context) ([]migration, error) {
	var migrations []migration
	rows, err := m.store.LoadObjects(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		if row := rows.Row(); row.Group == m.group {
			migrations = append(migrations, row)
		}
	}
	sort.Sort(migrationSorter(migrations))
	return migrations, rows.Err()
}

func (m *manager) getState(ctx context.Context, g MigrationGroup) ([]migrationState, error) {
	migrations := g.GetMigrations()
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}
	var result []migrationState
	it, jt := 0, 0
	for it < len(migrations) && jt < len(applied) {
		if migrations[it].Name < applied[jt].Name {
			result = append(result, migrationState{
				Name:      migrations[it].Name,
				Applied:   false,
				Supported: true,
			})
			it++
		} else if applied[jt].Name < migrations[it].Name {
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
			Name:      migrations[it].Name,
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

type MigrateOption func(state []migrationState, beginPos, endPos *int) error

func WithMigration(name string) MigrateOption {
	if name == "zero" {
		return WithZeroMigration
	}
	return func(state []migrationState, beginPos, endPos *int) error {
		for i := 0; i < len(state); i++ {
			if state[i].Name == name {
				*endPos = i + 1
				return nil
			}
		}
		return fmt.Errorf("invalid migration %q", name)
	}
}

func WithZeroMigration(state []migrationState, beginPos, endPos *int) error {
	*endPos = 0
	return nil
}

func WithFromMigration(name string) MigrateOption {
	return func(state []migrationState, beginPos, endPos *int) error {
		for i := 0; i < len(state); i++ {
			if state[i].Name == name {
				*beginPos = i
				return nil
			}
		}
		return fmt.Errorf("invalid migration %q", name)
	}
}

func (m *manager) Apply(ctx context.Context, g MigrationGroup, options ...MigrateOption) error {
	state, err := m.getState(ctx, g)
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
		if err := option(state, &beginPos, &endPos); err != nil {
			return err
		}
	}
	if endPos < beginPos {
		return m.applyBackward(ctx, g, state[endPos:beginPos])
	}
	return m.applyForward(ctx, g, state[beginPos:endPos])
}

func (m *manager) applyForward(ctx context.Context, g MigrationGroup, migrations []migrationState) error {
	if len(migrations) == 0 {
		log.Info("No migrations to apply: ", m.group)
		return nil
	}
	for _, mgr := range migrations {
		log.Info("Applying migration: ", m.group, ".", mgr.Name)
		impl := g.GetMigration(mgr.Name)
		if err := gosql.WrapTx(ctx, m.db.DB, func(tx *sql.Tx) error {
			ctx := WithTx(ctx, tx)
			// Apply migration.
			if err := impl.Apply(ctx, m.db); err != nil {
				return err
			}
			// Ignore update of migrations table with applied migration.
			if mgr.Applied {
				return nil
			}
			// Save to database that migration was applied.
			object := migration{
				Group:   m.group,
				Name:    mgr.Name,
				Version: config.Version,
				Time:    time.Now().Unix(),
			}
			return m.store.CreateObject(ctx, &object)
		}); err != nil {
			return err
		}
		log.Info("Migration applied: ", m.group, ".", mgr.Name)
	}
	return nil
}

func (m *manager) getAppliedMigration(ctx context.Context, group string, name string) (migration, error) {
	rows, err := m.store.FindObjects(ctx, gosql.Column("group").Equal(group).And(gosql.Column("name").Equal(name)))
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

func (m *manager) applyBackward(ctx context.Context, g MigrationGroup, migrations []migrationState) error {
	if len(migrations) == 0 {
		log.Info("No migrations to reverse apply: ", m.group)
		return nil
	}
	for i := 0; i < len(migrations); i++ {
		mgr := migrations[len(migrations)-i-1]
		log.Info("Reverse applying migration: ", m.group, ".", mgr.Name)
		impl := g.GetMigration(mgr.Name)
		if !mgr.Applied {
			return fmt.Errorf("migration %q is not applied", mgr.Name)
		}
		if err := gosql.WrapTx(ctx, m.db.DB, func(tx *sql.Tx) error {
			ctx := WithTx(ctx, tx)
			object, err := m.getAppliedMigration(ctx, m.group, mgr.Name)
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
		log.Info("Migration reverse applied: ", m.group, ".", mgr.Name)
	}
	return nil
}

type migration struct {
	ID      int64  `db:"id"`
	Group   string `db:"group"`
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

var mirgationTable = schema.CreateTable{
	Name: migrationTableName,
	Columns: []schema.Column{
		{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
		{Name: "group", Type: schema.String},
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

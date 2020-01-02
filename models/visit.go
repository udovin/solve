package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/udovin/solve/db"
)

// Visit represents user visit
type Visit struct {
	Id        int64  `db:"id" json:""`
	Time      int64  `db:"time" json:""`
	UserId    NInt64 `db:"user_id" json:",omitempty"`
	SessionId NInt64 `db:"session_id" json:",omitempty"`
}

// EventId returns id of visit
func (o Visit) EventId() int64 {
	return o.Id
}

// EventTime return time of visit
func (o Visit) EventTime() time.Time {
	return time.Unix(o.Time, 0)
}

// VisitManager represents visit manager
type VisitManager struct {
	db    *sql.DB
	store db.EventStore
}

// Create creates a new visit in the store
func (m *VisitManager) Create(visit Visit) (Visit, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return Visit{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if visit, err = m.CreateTx(tx, visit); err != nil {
		return Visit{}, err
	}
	if err := tx.Commit(); err != nil {
		return Visit{}, err
	}
	return visit, nil
}

// CreateTx creates a new visit in the store
func (m *VisitManager) CreateTx(tx *sql.Tx, visit Visit) (Visit, error) {
	event, err := m.store.CreateEvent(tx, visit)
	if err != nil {
		return Visit{}, err
	}
	return event.(Visit), nil
}

// InitTx does nothing
func (m *VisitManager) InitTx(tx *sql.Tx) error {
	return nil
}

// SyncTx does nothing
func (m *VisitManager) SyncTx(tx *sql.Tx) error {
	return nil
}

// MigrateTx upgrades database from current version
func (m *VisitManager) MigrateTx(tx *sql.Tx, version int) (int, error) {
	switch version {
	case 1:
		return 1, nil
	case 0:
		panic("implement me")
	default:
		return 0, fmt.Errorf("invalid version: %v", version)
	}
}

// NewVisitManager creates a new instance of ViewManager
func NewVisitManager(
	conn *sql.DB, table string, dialect db.Dialect,
) *VisitManager {
	return &VisitManager{
		db:    conn,
		store: db.NewEventStore(Visit{}, "id", table, dialect),
	}
}

package models

import (
	"database/sql"
	"fmt"
	"sync"
)

type ChangeType int8

const (
	CreateChange ChangeType = 1
	DeleteChange ChangeType = 2
	UpdateChange ChangeType = 3
)

type RowScan interface {
	Scan(dest ...interface{}) error
}

type ChangeBase struct {
	ID   int64      `db:"change_id"   json:""`
	Type ChangeType `db:"change_type" json:""`
	Time int64      `db:"change_time" json:""`
}

func (c *ChangeBase) ChangeID() int64 {
	return c.ID
}

func (c ChangeType) String() string {
	switch c {
	case CreateChange:
		return "Create"
	case DeleteChange:
		return "Delete"
	case UpdateChange:
		return "Update"
	default:
		return fmt.Sprintf("ChangeType(%d)", c)
	}
}

type Change interface {
	ChangeID() int64
}

// Store that supports change table
// Commonly used as in-memory cache for database table
type ChangeStore interface {
	// Get database connection
	GetDB() *sql.DB
	// Get change table name
	ChangeTableName() string
	// Scan change from result row
	scanChange(scan RowScan) (Change, error)
	// Save change to database
	saveChangeTx(tx *sql.Tx, change Change) error
	// Apply change to store
	applyChange(change Change)
}

type ChangeGap struct {
	beginID int64
	endID   int64
}

// Supports store consistency using change table
type ChangeManager struct {
	store        ChangeStore
	lastChangeID int64
	changeGaps   []ChangeGap
	lazyChanges  []Change
	mutex        sync.Mutex
}

func NewChangeManager(store ChangeStore) *ChangeManager {
	return &ChangeManager{store: store}
}

func (m *ChangeManager) Change(change Change) error {
	tx, err := m.store.GetDB().Begin()
	if err != nil {
		return err
	}
	err = m.ChangeTx(tx, change)
	if err != nil {
		_ = tx.Rollback()
		m.Reset()
		return err
	}
	if err := tx.Commit(); err != nil {
		m.Reset()
		return err
	}
	m.Push()
	return nil
}

func (m *ChangeManager) ChangeTx(tx *sql.Tx, change Change) error {
	if err := m.SyncTx(tx); err != nil {
		return err
	}
	if err := m.store.saveChangeTx(tx, change); err != nil {
		return err
	}
	m.lazyChanges = append(m.lazyChanges, change)
	return nil
}

func (m *ChangeManager) Sync() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	rows, err := m.store.GetDB().Query(
		fmt.Sprintf(
			`SELECT * FROM "%s" WHERE "change_id" > $1 ORDER BY "change_id"`,
			m.store.ChangeTableName(),
		),
		m.lastChangeID,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			panic(err)
		}
	}()
	for rows.Next() {
		change, err := m.store.scanChange(rows)
		if err != nil {
			return err
		}
		m.applyChange(change)
	}
	return nil
}

func (m *ChangeManager) SyncTx(tx *sql.Tx) error {
	rows, err := tx.Query(
		fmt.Sprintf(
			`SELECT * FROM "%s" WHERE "change_id" > $1 ORDER BY "change_id"`,
			m.store.ChangeTableName(),
		),
		m.lastChangeID,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			panic(err)
		}
	}()
	for rows.Next() {
		change, err := m.store.scanChange(rows)
		if err != nil {
			return err
		}
		m.applyChange(change)
	}
	return nil
}

// Apply change to store and increase change id
func (m *ChangeManager) applyChange(change Change) {
	m.store.applyChange(change)
	if m.lastChangeID >= change.ChangeID() {
		return
	}
	if m.lastChangeID+1 < change.ChangeID() {
		m.changeGaps = append(m.changeGaps, ChangeGap{
			beginID: m.lastChangeID + 1,
			endID:   change.ChangeID(),
		})
	}
	m.lastChangeID = change.ChangeID()
}

func (m *ChangeManager) Push() {
	for _, change := range m.lazyChanges {
		m.applyChange(change)
	}
	m.Reset()
}

func (m *ChangeManager) Reset() {
	m.lazyChanges = nil
}

func (m *ChangeManager) LockTx(tx *sql.Tx) error {
	_, err := tx.Exec(
		fmt.Sprintf(`LOCK TABLE "%s"`, m.store.ChangeTableName()),
	)
	return err
}

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
	saveChangeTx(tx *ChangeTx, change Change) error
	// Apply change to store
	applyChange(change Change)
}

type ChangeGap struct {
	beginID int64
	endID   int64
}

type ChangeTx struct {
	*sql.Tx
	changes map[*ChangeManager][]Change
}

// Supports store consistency using change table
type ChangeManager struct {
	store        ChangeStore
	lastChangeID int64
	changeGaps   []ChangeGap
	mutex        sync.Mutex
}

func NewChangeManager(store ChangeStore) *ChangeManager {
	return &ChangeManager{store: store}
}

func (tx *ChangeTx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return err
	}
	for manager, changes := range tx.changes {
		manager.mutex.Lock()
		for _, change := range changes {
			manager.applyChange(change)
		}
		manager.mutex.Unlock()
		delete(tx.changes, manager)
	}
	return nil
}

func (tx *ChangeTx) Rollback() error {
	if err := tx.Tx.Rollback(); err != nil {
		return err
	}
	for manager := range tx.changes {
		delete(tx.changes, manager)
	}
	return nil
}

func (m *ChangeManager) Begin() (*ChangeTx, error) {
	tx, err := m.store.GetDB().Begin()
	if err != nil {
		return nil, err
	}
	return &ChangeTx{
		Tx: tx, changes: make(map[*ChangeManager][]Change),
	}, nil
}

func (m *ChangeManager) Change(change Change) error {
	tx, err := m.Begin()
	if err != nil {
		return err
	}
	if err := m.ChangeTx(tx, change); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (m *ChangeManager) ChangeTx(tx *ChangeTx, change Change) error {
	if err := m.SyncTx(tx); err != nil {
		return err
	}
	if err := m.store.saveChangeTx(tx, change); err != nil {
		return err
	}
	tx.changes[m] = append(tx.changes[m], change)
	return nil
}

func (m *ChangeManager) Sync() error {
	tx, err := m.Begin()
	if err != nil {
		return err
	}
	err = m.SyncTx(tx)
	_ = tx.Rollback()
	return err
}

func (m *ChangeManager) SyncTx(tx *ChangeTx) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	rows, err := tx.Query(
		fmt.Sprintf(
			`SELECT * FROM "%s" `+
				`WHERE "change_id" > $1 ORDER BY "change_id"`,
			m.store.ChangeTableName(),
		),
		m.lastChangeID,
	)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
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

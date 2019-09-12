package models

import (
	"container/list"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"math"
	"sync"

	"github.com/mattn/go-sqlite3"
)

// ChangeType identifies BaseChange type
type ChangeType int8

const (
	// CreateChange should be used for create objects
	CreateChange ChangeType = 1
	// DeleteChange should be used for delete objects
	DeleteChange ChangeType = 2
	// UpdateChange should be used for update objects
	UpdateChange ChangeType = 3
)

// Scanner should scan into specified destinations
type Scanner interface {
	// Scan scans into specified destinations
	Scan(dest ...interface{}) error
}

// BaseChange contains columns for typical change records
type BaseChange struct {
	ID   int64      `json:"" db:"change_id"`
	Type ChangeType `json:"" db:"change_type"`
	Time int64      `json:"" db:"change_time"`
}

// ChangeID returns change identifier
func (c *BaseChange) ChangeID() int64 {
	return c.ID
}

// String returns string representation of change type
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

// Change is an object that has ChangeID
type Change interface {
	// ChangeID should return index of change
	ChangeID() int64
}

// ChangeStore is a store that supports change table
// Commonly used as in-memory cache for database table
type ChangeStore interface {
	// GetLocker should return write locker
	GetLocker() sync.Locker
	// InitChanges should initialize change store
	InitChanges(tx *sql.Tx) (int64, error)
	// LoadChanges should load changes from gap
	LoadChanges(tx *sql.Tx, gap ChangeGap) (*sql.Rows, error)
	// ScanChange should scan change from result row
	ScanChange(scan Scanner) (Change, error)
	// SaveChange should save change to database
	SaveChange(tx *sql.Tx, change Change) error
	// ApplyChange should apply change to store
	ApplyChange(change Change)
}

// ChangeGap stores change gap range for specified change store
type ChangeGap struct {
	BeginID int64
	EndID   int64
}

// ChangeTx stores non applied changes for current transaction
type ChangeTx struct {
	*sql.Tx
	changes map[*ChangeManager][]Change
	updated map[*ChangeManager]bool
}

// ChangeManager supports store consistency using change table
//
// TODO: Replace list with Binary Search Tree
type ChangeManager struct {
	// Store for manager
	store ChangeStore
	// Connection to database
	db *sql.DB
	// Change gaps are required for allow transactions without
	// locking full change table
	changeGaps   *list.List
	lastChangeID int64
	mutex        sync.Mutex
}

// NewChangeManager creates new instance of ChangeManager
func NewChangeManager(store ChangeStore, db *sql.DB) *ChangeManager {
	return &ChangeManager{
		store:      store,
		db:         db,
		changeGaps: list.New(),
	}
}

// Commit applies changes to all change managers
func (tx *ChangeTx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return err
	}
	for manager, changes := range tx.changes {
		func() {
			manager.mutex.Lock()
			defer manager.mutex.Unlock()
			for _, change := range changes {
				manager.applyChange(change)
			}
		}()
		delete(tx.changes, manager)
	}
	return nil
}

// Rollback removes non applied changes from all change managers
func (tx *ChangeTx) Rollback() error {
	if err := tx.Tx.Rollback(); err != nil {
		return err
	}
	for manager := range tx.changes {
		delete(tx.changes, manager)
	}
	return nil
}

// Init initializes change manager an change store
func (m *ChangeManager) Init() error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	id, err := m.store.InitChanges(tx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	m.lastChangeID = id
	return nil
}

// Begin starts new transaction
func (m *ChangeManager) Begin() (*ChangeTx, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	return &ChangeTx{
		Tx:      tx,
		changes: make(map[*ChangeManager][]Change),
		updated: make(map[*ChangeManager]bool),
	}, nil
}

// Change immediately applies change to change store
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

// ChangeTx adds change of change store to transaction
func (m *ChangeManager) ChangeTx(tx *ChangeTx, change Change) error {
	if err := m.SyncTx(tx); err != nil {
		return err
	}
	if err := m.store.SaveChange(tx.Tx, change); err != nil {
		return err
	}
	tx.changes[m] = append(tx.changes[m], change)
	tx.updated[m] = true
	return nil
}

// Sync loads new changes from change table
func (m *ChangeManager) Sync() error {
	tx, err := m.Begin()
	if err != nil {
		return err
	}
	err = m.SyncTx(tx)
	_ = tx.Rollback()
	return err
}

// Some transactions may failure and such gaps will never been removed
// so we should skip this gaps after some other changes
const changeGapSkipWindow = 5000

// SyncTx syncs store with change table in transaction
func (m *ChangeManager) SyncTx(tx *ChangeTx) error {
	if tx.updated[m] {
		return nil
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.skipOldChangeGaps()
	if err := m.loadChangeGaps(tx); err != nil {
		return err
	}
	return m.loadNewChanges(tx)
}

// skipOldGaps removes old gaps from change manager
func (m *ChangeManager) skipOldChangeGaps() {
	for e := m.changeGaps.Front(); e != nil; {
		curr := e.Value.(ChangeGap)
		if curr.EndID+changeGapSkipWindow >= m.lastChangeID {
			break
		}
		next := e.Next()
		m.changeGaps.Remove(e)
		e = next
	}
}

// loadChangeGaps loads changes from change table for existing gaps
func (m *ChangeManager) loadChangeGaps(tx *ChangeTx) error {
	for e := m.changeGaps.Front(); e != nil; {
		curr := e.Value.(ChangeGap)
		rows, err := m.store.LoadChanges(tx.Tx, curr)
		if err != nil {
			return err
		}
		for rows.Next() {
			change, err := m.store.ScanChange(rows)
			if err != nil {
				_ = rows.Close()
				return err
			}
			if change.ChangeID() < curr.BeginID {
				_ = rows.Close()
				panic("ChangeID should be not less than gap BeginID")
			}
			if change.ChangeID() >= curr.EndID {
				_ = rows.Close()
				panic("ChangeID should be less than gap EndID")
			}
			m.applyStoreChange(change)
			next := ChangeGap{
				BeginID: change.ChangeID() + 1,
				EndID:   curr.EndID,
			}
			if curr.BeginID < change.ChangeID() {
				curr.EndID = change.ChangeID()
				e.Value = curr
				if next.BeginID < next.EndID {
					e = m.changeGaps.InsertAfter(next, e)
					curr = next
				}
			} else {
				curr.BeginID++
				if curr.BeginID >= curr.EndID {
					next := e.Next()
					m.changeGaps.Remove(e)
					e = next
					continue
				}
				e.Value = curr
			}
		}
		_ = rows.Close()
		e = e.Next()
	}
	return nil
}

// loadNewChanges loads new changes and applies them to store
//
// TODO(iudovin): Make normal code for non existent right border
func (m *ChangeManager) loadNewChanges(tx *ChangeTx) error {
	rows, err := m.store.LoadChanges(tx.Tx, ChangeGap{
		BeginID: m.lastChangeID + 1,
		EndID:   math.MaxInt32,
	})
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Println("Error:", err)
		}
	}()
	for rows.Next() {
		change, err := m.store.ScanChange(rows)
		if err != nil {
			return err
		}
		m.applyChange(change)
	}
	return nil
}

// applyChange applies change to store and increase change id
func (m *ChangeManager) applyChange(change Change) {
	if change.ChangeID() <= m.lastChangeID {
		for e := m.changeGaps.Front(); e != nil; e = e.Next() {
			curr := e.Value.(ChangeGap)
			if change.ChangeID() >= curr.EndID {
				continue
			}
			if change.ChangeID() < curr.BeginID {
				break
			}
			m.applyStoreChange(change)
			next := ChangeGap{
				BeginID: change.ChangeID() + 1,
				EndID:   curr.EndID,
			}
			if curr.BeginID < change.ChangeID() {
				curr.EndID = change.ChangeID()
				e.Value = curr
				if next.BeginID < next.EndID {
					e = m.changeGaps.InsertAfter(next, e)
				}
			} else {
				curr.BeginID++
				e.Value = curr
				if curr.BeginID >= curr.EndID {
					m.changeGaps.Remove(e)
				}
			}
			return
		}
		panic("Change ID should be greater than last ChangeID")
	}
	m.applyStoreChange(change)
	if m.lastChangeID+1 < change.ChangeID() {
		_ = m.changeGaps.PushBack(ChangeGap{
			BeginID: m.lastChangeID + 1,
			EndID:   change.ChangeID(),
		})
	}
	m.lastChangeID = change.ChangeID()
}

// applyStoreChange safely applies change to store
func (m *ChangeManager) applyStoreChange(change Change) {
	locker := m.store.GetLocker()
	locker.Lock()
	defer locker.Unlock()
	m.store.ApplyChange(change)
}

func execTxReturningID(
	driver driver.Driver, tx *sql.Tx, query, name string, args ...interface{},
) (id int64, err error) {
	if _, ok := driver.(*sqlite3.SQLiteDriver); ok {
		return execSQLiteTxReturningID(tx, query, args...)
	}
	return execPostgresTxReturningID(tx, query, name, args...)
}

func execSQLiteTxReturningID(
	tx *sql.Tx, query string, args ...interface{},
) (id int64, err error) {
	res, err := tx.Exec(query, args...)
	if err != nil {
		return
	}
	id, err = res.LastInsertId()
	return
}

func execPostgresTxReturningID(
	tx *sql.Tx, query, name string, args ...interface{},
) (id int64, err error) {
	err = tx.QueryRow(
		fmt.Sprintf("%s RETURNING %q", query, name),
		args...,
	).Scan(&id)
	return
}

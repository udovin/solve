package models

import (
	"container/list"
	"database/sql"
	"fmt"
	"math"
	"sync"
)

type ChangeType int8

const (
	CreateChange ChangeType = 1
	DeleteChange ChangeType = 2
	UpdateChange ChangeType = 3
)

// Record scanner
type Scanner interface {
	Scan(dest ...interface{}) error
}

// Base columns for typical change records
type BaseChange struct {
	ID   int64      `json:"" db:"change_id"`
	Type ChangeType `json:"" db:"change_type"`
	Time int64      `json:"" db:"change_time"`
}

// Get change identifier
func (c *BaseChange) ChangeID() int64 {
	return c.ID
}

// Get string representation of change type
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
	// Get write locker
	getLocker() sync.Locker
	// Setup changes
	setupChanges(tx *sql.Tx) (int64, error)
	// Load changes from gap
	loadChangeGapTx(tx *sql.Tx, gap ChangeGap) (*sql.Rows, error)
	// Scan change from result row
	scanChange(scan Scanner) (Change, error)
	// Save change to database
	saveChangeTx(tx *sql.Tx, change Change) error
	// Apply change to store
	applyChange(change Change)
}

type ChangeGap struct {
	BeginID int64
	EndID   int64
}

type ChangeTx struct {
	*sql.Tx
	changes map[*ChangeManager][]Change
}

// Supports store consistency using change table
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
}

func NewChangeManager(store ChangeStore, db *sql.DB) *ChangeManager {
	return &ChangeManager{
		store:      store,
		db:         db,
		changeGaps: list.New(),
	}
}

func (tx *ChangeTx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return err
	}
	for manager, changes := range tx.changes {
		locker := manager.store.getLocker()
		locker.Lock()
		for _, change := range changes {
			manager.applyChange(change)
		}
		locker.Unlock()
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

func (m *ChangeManager) Setup() error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	id, err := m.store.setupChanges(tx)
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

func (m *ChangeManager) Begin() (*ChangeTx, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	return &ChangeTx{
		Tx:      tx,
		changes: make(map[*ChangeManager][]Change),
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
	if err := m.store.saveChangeTx(tx.Tx, change); err != nil {
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

const changeGapSkipWindow = 5000

func (m *ChangeManager) SyncTx(tx *ChangeTx) error {
	locker := m.store.getLocker()
	locker.Lock()
	defer locker.Unlock()
	for e := m.changeGaps.Front(); e != nil; {
		curr := e.Value.(ChangeGap)
		if curr.EndID+changeGapSkipWindow >= m.lastChangeID {
			break
		}
		next := e.Next()
		m.changeGaps.Remove(e)
		e = next
	}
	for e := m.changeGaps.Front(); e != nil; {
		curr := e.Value.(ChangeGap)
		rows, err := m.store.loadChangeGapTx(tx.Tx, curr)
		if err != nil {
			return err
		}
		for rows.Next() {
			change, err := m.store.scanChange(rows)
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
			m.store.applyChange(change)
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
	rows, err := m.store.loadChangeGapTx(tx.Tx, ChangeGap{
		BeginID: m.lastChangeID + 1,
		EndID:   math.MaxInt64,
	})
	if err != nil {
		return err
	}
	for rows.Next() {
		change, err := m.store.scanChange(rows)
		if err != nil {
			_ = rows.Close()
			return err
		}
		m.applyChange(change)
	}
	_ = rows.Close()
	return nil
}

// Apply change to store and increase change id
func (m *ChangeManager) applyChange(change Change) {
	if change.ChangeID() <= m.lastChangeID {
		for e := m.changeGaps.Front(); e != nil; e = e.Next() {
			curr := e.Value.(ChangeGap)
			if change.ChangeID() < curr.BeginID {
				continue
			}
			if change.ChangeID() >= curr.EndID {
				break
			}
			m.store.applyChange(change)
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
				e.Value = curr
				if curr.BeginID >= curr.EndID {
					m.changeGaps.Remove(e)
				}
			}
			return
		}
		panic("Change ID should be greater than last ChangeID")
	}
	m.store.applyChange(change)
	if m.lastChangeID+1 < change.ChangeID() {
		_ = m.changeGaps.PushBack(ChangeGap{
			BeginID: m.lastChangeID + 1,
			EndID:   change.ChangeID(),
		})
	}
	m.lastChangeID = change.ChangeID()
}

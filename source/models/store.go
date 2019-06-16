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

type Change interface {
	ChangeID() int64
	ChangeType() ChangeType
	ChangeData() interface{}
}

type Store interface {
	GetDB() *sql.DB
	TableName() string
}

type ChangeStore interface {
	GetDB() *sql.DB
	ChangeTableName() string
	scanChange(scan RowScan) (Change, error)
	applyChange(change Change) error
}

type ChangeManager struct {
	store        ChangeStore
	lastChangeID int64
	mutex        sync.Mutex
}

func NewChangeManager(store ChangeStore) *ChangeManager {
	return &ChangeManager{store: store}
}

func (m *ChangeManager) LockTx(tx *sql.Tx) error {
	_, err := tx.Exec(
		fmt.Sprintf(`LOCK TABLE "%s"`, m.store.ChangeTableName()),
	)
	return err
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
		if err := m.store.applyChange(change); err != nil {
			return err
		}
		m.lastChangeID = change.ChangeID()
	}
	return nil
}

func (m *ChangeManager) SyncTx(tx *sql.Tx) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
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
		if err := m.store.applyChange(change); err != nil {
			return err
		}
		m.lastChangeID = change.ChangeID()
	}
	return nil
}

func (m *ChangeManager) Create(data interface{}) error {
	tx, err := m.store.GetDB().Begin()
	if err != nil {
		return err
	}
	if err := m.CreateTx(tx, data); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (m *ChangeManager) CreateTx(tx *sql.Tx, data interface{}) error {
	if err := m.LockTx(tx); err != nil {
		return err
	}
	if err := m.SyncTx(tx); err != nil {
		return err
	}
	if _, ok := m.store.(Store); ok {
	}
	return nil
}

func (m *ChangeManager) Update(data interface{}) error {
	tx, err := m.store.GetDB().Begin()
	if err != nil {
		return err
	}
	if err := m.UpdateTx(tx, data); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (m *ChangeManager) UpdateTx(tx *sql.Tx, data interface{}) error {
	if err := m.LockTx(tx); err != nil {
		return err
	}
	if err := m.SyncTx(tx); err != nil {
		return err
	}
	if _, ok := m.store.(Store); ok {
	}
	return nil
}

func (m *ChangeManager) Delete(data interface{}) error {
	tx, err := m.store.GetDB().Begin()
	if err != nil {
		return err
	}
	if err := m.DeleteTx(tx, data); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (m *ChangeManager) DeleteTx(tx *sql.Tx, data interface{}) error {
	if err := m.LockTx(tx); err != nil {
		return err
	}
	if err := m.SyncTx(tx); err != nil {
		return err
	}
	if _, ok := m.store.(Store); ok {
	}
	return nil
}

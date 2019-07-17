package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/config"
)

type StoreMock struct {
	db *sql.DB
}

type ChangeMock struct {
	BaseChange
}

func (s *StoreMock) GetDB() *sql.DB {
	return nil
}

func (s *StoreMock) ChangeTableName() string {
	return "test_solve_store_mock"
}

func (s *StoreMock) loadChangeGapTx(
	tx *ChangeTx, gap ChangeGap,
) (*sql.Rows, error) {
	return nil, nil
}

func (s *StoreMock) scanChange(scan Scanner) (Change, error) {
	return nil, nil
}

func (s *StoreMock) saveChangeTx(tx *ChangeTx, change Change) error {
	return nil
}

func (s *StoreMock) applyChange(change Change) {

}

func TestChangeManager(t *testing.T) {
	t.Skip("Broken test")
	cfg := config.DatabaseConfig{
		Driver:  config.SQLiteDriver,
		Options: config.SQLiteOptions{Path: "?mode=memory"},
	}
	db, err := cfg.CreateDB()
	if err != nil {
		t.Error(err)
	}
	store := StoreMock{db: db}
	manager := NewChangeManager(&store)
	if err := manager.Change(&ChangeMock{}); err != nil {
		t.Error("Error: ", err)
	}
}

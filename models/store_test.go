package models

import (
	"database/sql"
	"testing"
)

type StoreMock struct{}

type ChangeMock struct {
	ChangeBase
}

func (s *StoreMock) GetDB() *sql.DB {
	return nil
}

func (s *StoreMock) ChangeTableName() string {
	return "test_solve_store_mock"
}

func (s *StoreMock) scanChange(scan RowScan) (Change, error) {
	return nil, nil
}

func (s *StoreMock) saveChangeTx(tx *ChangeTx, change Change) error {
	return nil
}

func (s *StoreMock) applyChange(change Change) {

}

func TestChangeManager(t *testing.T) {
	t.Skip("Broken test")
	store := StoreMock{}
	manager := NewChangeManager(&store)
	if err := manager.Change(&ChangeMock{}); err != nil {
		t.Error("Error: ", err)
	}
}

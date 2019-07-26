package models

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"
)

type Mock struct {
	ID    int    `db:"id"`
	Value string `db:"value"`
}

type MockStore struct {
	mocks map[int]Mock
	mutex sync.RWMutex
}

type mockChange struct {
	BaseChange
	Mock
}

func (s *MockStore) getLocker() sync.Locker {
	return &s.mutex
}

func (s *MockStore) Get(id int) (Mock, bool) {
	mock, ok := s.mocks[id]
	return mock, ok
}

func (s *MockStore) setupChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *MockStore) loadChangeGapTx(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		`SELECT `+
			`"change_id", "change_type", "change_time", "id", "value"`+
			` FROM "test_mock_change"`+
			` WHERE "change_id" >= $1 AND "change_id" < $2`+
			` ORDER BY "change_id"`,
		gap.BeginID, gap.EndID,
	)
}

func (s *MockStore) scanChange(scan Scanner) (Change, error) {
	change := &mockChange{}
	err := scan.Scan(
		&change.BaseChange.ID, &change.Type, &change.Time,
		&change.Mock.ID, &change.Value,
	)
	return change, err
}

func (s *MockStore) saveChangeTx(tx *sql.Tx, change Change) error {
	mock := change.(*mockChange)
	mock.Time = time.Now().Unix()
	res, err := tx.Exec(
		`INSERT INTO "test_mock_change"`+
			` ("change_type", "change_time", "id", "value")`+
			` VALUES ($1, $2, $3, $4)`,
		mock.Type, mock.Time, mock.Mock.ID, mock.Value,
	)
	if err != nil {
		return err
	}
	mock.BaseChange.ID, err = res.LastInsertId()
	return err
}

func (s *MockStore) applyChange(change Change) {
	mock := change.(*mockChange)
	switch mock.Type {
	case UpdateChange:
		fallthrough
	case CreateChange:
		s.mocks[mock.Mock.ID] = mock.Mock
	case DeleteChange:
		delete(s.mocks, mock.Mock.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			mock.Type,
		))
	}
}

func TestChangeManager(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := MockStore{mocks: make(map[int]Mock)}
	manager := NewChangeManager(&store, db)
	mocks := []Mock{
		{ID: 1, Value: "hello"},
		{ID: 2, Value: "golang"},
		{ID: 3, Value: "solve"},
		{ID: 4, Value: "model"},
	}
	for _, mock := range mocks {
		if err := manager.Change(&mockChange{
			BaseChange: BaseChange{Type: CreateChange},
			Mock:       mock,
		}); err != nil {
			t.Error("Error: ", err)
		}
	}
	for _, mock := range mocks {
		m, ok := store.Get(mock.ID)
		if !ok {
			t.Errorf("Mock with ID = %d does not exist", mock.ID)
		}
		if m.Value != mock.Value {
			t.Errorf(
				"Expected '%s' but found '%s'",
				mock.Value, m.Value,
			)
		}
	}
}

func TestChangeManager_applyChange(t *testing.T) {
	store := MockStore{mocks: make(map[int]Mock)}
	manager := NewChangeManager(&store, db)
	applyChange := func(id int64) {
		manager.applyChange(&mockChange{
			BaseChange{id, CreateChange, 0},
			Mock{int(id), fmt.Sprintf("%d", id)},
		})
	}
	checkGapsLen := func(l int) {
		if manager.changeGaps.Len() != l {
			t.Errorf(
				"Expected len = %d, but found %d",
				l, manager.changeGaps.Len(),
			)
		}
	}
	applyChange(3)
	checkGapsLen(1)
	applyChange(1)
	checkGapsLen(1)
	applyChange(2)
	checkGapsLen(0)
	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Error("Panic expected")
			}
		}()
		store.applyChange(&BaseChange{})
	}()
	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Error("Panic expected")
			}
		}()
		store.applyChange(nil)
	}()
}

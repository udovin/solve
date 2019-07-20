package models

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
)

type Mock struct {
	ID    int    `db:"id"`
	Value string `db:"value"`
}

type MockStore struct {
	db    *sql.DB
	mocks map[int]Mock
}

type MockChange struct {
	BaseChange
	Mock
}

func (s *MockStore) GetDB() *sql.DB {
	return s.db
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
	change := &MockChange{}
	err := scan.Scan(
		&change.BaseChange.ID, &change.Type, &change.Time,
		&change.Mock.ID, &change.Value,
	)
	return change, err
}

func (s *MockStore) saveChangeTx(tx *sql.Tx, change Change) error {
	mock := change.(*MockChange)
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
	mockChange := change.(*MockChange)
	switch mockChange.Type {
	case UpdateChange:
		fallthrough
	case CreateChange:
		s.mocks[mockChange.Mock.ID] = mockChange.Mock
	case DeleteChange:
		delete(s.mocks, mockChange.Mock.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			mockChange.Type,
		))
	}
}

func TestChangeManager(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := MockStore{db: db, mocks: make(map[int]Mock)}
	manager := NewChangeManager(&store)
	mocks := []Mock{
		{ID: 1, Value: "hello"},
		{ID: 2, Value: "golang"},
		{ID: 3, Value: "solve"},
		{ID: 4, Value: "model"},
	}
	for _, mock := range mocks {
		if err := manager.Change(&MockChange{
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
	store := MockStore{db: db, mocks: make(map[int]Mock)}
	manager := NewChangeManager(&store)
	applyChange := func(id int64) {
		manager.applyChange(&MockChange{
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
}

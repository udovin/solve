package models

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/udovin/solve/config"
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

func (s *MockStore) loadChangeGapTx(
	tx *ChangeTx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		`SELECT `+
			`"change_id", "change_type", "change_time", "id", "value"`+
			` FROM "mock_change"`+
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

func (s *MockStore) saveChangeTx(tx *ChangeTx, change Change) error {
	mock := change.(*MockChange)
	mock.Time = time.Now().Unix()
	res, err := tx.Exec(
		`INSERT INTO "mock_change"`+
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

var db *sql.DB

func setup() {
	cfg := config.DatabaseConfig{
		Driver:  config.SQLiteDriver,
		Options: config.SQLiteOptions{Path: "?mode=memory"},
	}
	var err error
	db, err = cfg.CreateDB()
	if err != nil {
		os.Exit(1)
	}
	_, err = db.Exec(
		`CREATE TABLE "mock_change"` +
			` ("change_id" INTEGER PRIMARY KEY,` +
			` "change_type" INT8,` +
			` "change_time" BIGINT,` +
			` "id" INTEGER,` +
			` "value" VARCHAR(255))`,
	)
	if err != nil {
		os.Exit(1)
	}
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func TestChangeManager(t *testing.T) {
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

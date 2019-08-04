package models

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"
)

type Fake struct {
	ID    int    `db:"id"`
	Value string `db:"value"`
}

type FakeStore struct {
	fakes map[int]Fake
	mutex sync.RWMutex
}

type fakeChange struct {
	BaseChange
	Fake
}

func (s *FakeStore) getLocker() sync.Locker {
	return &s.mutex
}

func (s *FakeStore) Get(id int) (Fake, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	mock, ok := s.fakes[id]
	return mock, ok
}

func (s *FakeStore) initChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *FakeStore) loadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		`SELECT `+
			`"change_id", "change_type", "change_time", "id", "value"`+
			` FROM "test_fake_change"`+
			` WHERE "change_id" >= $1 AND "change_id" < $2`+
			` ORDER BY "change_id"`,
		gap.BeginID, gap.EndID,
	)
}

func (s *FakeStore) scanChange(scan Scanner) (Change, error) {
	change := &fakeChange{}
	err := scan.Scan(
		&change.BaseChange.ID, &change.Type, &change.Time,
		&change.Fake.ID, &change.Value,
	)
	return change, err
}

func (s *FakeStore) saveChange(tx *sql.Tx, change Change) error {
	mock := change.(*fakeChange)
	mock.Time = time.Now().Unix()
	res, err := tx.Exec(
		`INSERT INTO "test_fake_change"`+
			` ("change_type", "change_time", "id", "value")`+
			` VALUES ($1, $2, $3, $4)`,
		mock.Type, mock.Time, mock.Fake.ID, mock.Value,
	)
	if err != nil {
		return err
	}
	mock.BaseChange.ID, err = res.LastInsertId()
	return err
}

func (s *FakeStore) applyChange(change Change) {
	mock := change.(*fakeChange)
	switch mock.Type {
	case UpdateChange:
		fallthrough
	case CreateChange:
		s.fakes[mock.Fake.ID] = mock.Fake
	case DeleteChange:
		delete(s.fakes, mock.Fake.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			mock.Type,
		))
	}
}

var fakes = []Fake{
	{ID: 1, Value: "hello"},
	{ID: 2, Value: "golang"},
	{ID: 3, Value: "solve"},
	{ID: 4, Value: "model"},
}

func TestChangeManager(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := FakeStore{fakes: make(map[int]Fake)}
	manager := NewChangeManager(&store, db)
	for _, fake := range fakes {
		if err := manager.Change(&fakeChange{
			BaseChange: BaseChange{Type: CreateChange},
			Fake:       fake,
		}); err != nil {
			t.Error("Error: ", err)
		}
	}
	for _, fake := range fakes {
		m, ok := store.Get(fake.ID)
		if !ok {
			t.Errorf("Fake with ID = %d does not exist", fake.ID)
		}
		if m.Value != fake.Value {
			t.Errorf(
				"Expected '%s' but found '%s'",
				fake.Value, m.Value,
			)
		}
	}
}

func TestChangeManager_applyChange(t *testing.T) {
	store := FakeStore{fakes: make(map[int]Fake)}
	manager := NewChangeManager(&store, db)
	applyChange := func(id int64) {
		manager.applyChange(&fakeChange{
			BaseChange{id, CreateChange, 0},
			Fake{int(id), fmt.Sprintf("%d", id)},
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
	for i := int64(11); i <= 20; i++ {
		applyChange(i)
		checkGapsLen(1)
	}
	for i := int64(4); i < 10; i++ {
		applyChange(i)
		checkGapsLen(1)
	}
	applyChange(10)
	checkGapsLen(0)
	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Error("Panic expected")
			}
		}()
		applyChange(5)
	}()
}

func TestBaseChange_ChangeID(t *testing.T) {
	createChange := CreateChange
	if createChange.String() != "Create" {
		t.Error("Create change has invalid string representation")
	}
	updateChange := UpdateChange
	if updateChange.String() != "Update" {
		t.Error("Update change has invalid string representation")
	}
	deleteChange := DeleteChange
	if deleteChange.String() != "Delete" {
		t.Error("Delete change has invalid string representation")
	}
	unknownChange := ChangeType(127)
	if unknownChange.String() != "ChangeType(127)" {
		t.Error("Unknown change has invalid string representation")
	}
}

func TestChangeManager_Sync(t *testing.T) {
	setup(t)
	defer teardown(t)
	store1 := FakeStore{fakes: make(map[int]Fake)}
	manager1 := NewChangeManager(&store1, db)
	store2 := FakeStore{fakes: make(map[int]Fake)}
	manager2 := NewChangeManager(&store2, db)
	for i, fake := range fakes {
		if err := manager1.Change(&fakeChange{
			BaseChange: BaseChange{ID: int64(i), Type: CreateChange},
			Fake:       fake,
		}); err != nil {
			t.Error(err)
		}
	}
	for _, fake := range fakes {
		if f, ok := store1.Get(fake.ID); !ok || f != fake {
			t.Error("Invalid value")
		}
	}
	for _, fake := range fakes {
		if _, ok := store2.Get(fake.ID); ok {
			t.Error("Store does not have items")
		}
	}
	if err := manager2.Sync(); err != nil {
		t.Error(err)
	}
	for _, fake := range fakes {
		if f, ok := store2.Get(fake.ID); !ok || f != fake {
			t.Error("Invalid value")
		}
	}
}

func TestChangeManager_SyncClosed(t *testing.T) {
	setup(t)
	teardown(t)
	store := FakeStore{fakes: make(map[int]Fake)}
	manager := NewChangeManager(&store, db)
	if err := manager.Sync(); err == nil {
		t.Error("Expected sync error")
	}
}

func TestChangeManager_ChangeClosed(t *testing.T) {
	setup(t)
	store := FakeStore{fakes: make(map[int]Fake)}
	manager := NewChangeManager(&store, db)
	tx, err := manager.Begin()
	if err == nil {
		err = tx.Rollback()
	}
	teardown(t)
	if err != nil {
		t.Error(err)
	}
	if err := manager.ChangeTx(tx, &fakeChange{
		BaseChange: BaseChange{ID: 1, Type: CreateChange},
		Fake:       Fake{ID: 1, Value: "Fake item"},
	}); err == nil {
		t.Error("Expected sync error")
	}
}

func BenchmarkChangeManager_Change(b *testing.B) {
	setup(b)
	defer teardown(b)
	store := FakeStore{fakes: make(map[int]Fake)}
	manager := NewChangeManager(&store, db)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := manager.Change(&fakeChange{
			BaseChange: BaseChange{Type: CreateChange},
			Fake:       Fake{ID: i + 1, Value: "Value"},
		}); err != nil {
			b.Error("Error: ", err)
		}
	}
}

package models

import (
	"testing"
	"time"
)

func TestContestStore_getLocker(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewContestStore(db, "test_contest", "test_contest_change")
	if store.GetLocker() == nil {
		t.Fatal("locker should not be nil")
	}
}

func TestContestStore_applyChange(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewContestStore(db, "test_contest", "test_contest_change")
	store.ApplyChange(&contestChange{
		BaseChange: BaseChange{ID: 1, Type: CreateChange, Time: 0},
		Contest:    Contest{ID: 1, OwnerID: 1},
	})
	m, ok := store.Get(1)
	if !ok {
		t.Fatal("Contest should exists")
	}
	if m.OwnerID != 1 {
		t.Fatal("Wrong owner ID")
	}
	store.ApplyChange(&contestChange{
		BaseChange: BaseChange{ID: 2, Type: UpdateChange, Time: 1},
		Contest:    Contest{ID: 1, OwnerID: 2},
	})
	m, ok = store.Get(1)
	if !ok {
		t.Fatal("Contest should exists")
	}
	if m.OwnerID != 2 {
		t.Fatal("Wrong owner ID")
	}
	store.ApplyChange(&contestChange{
		BaseChange: BaseChange{ID: 3, Type: DeleteChange, Time: 2},
		Contest:    Contest{ID: 1},
	})
	if _, ok := store.Get(1); ok {
		t.Fatal("Contest should be deleted")
	}
	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Fatal("Panic expected")
			}
		}()
		store.ApplyChange(&contestChange{
			BaseChange: BaseChange{ID: 4, Type: ChangeType(126), Time: 0},
			Contest:    Contest{ID: 2, OwnerID: 1},
		})
	}()
	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Fatal("Panic expected")
			}
		}()
		store.ApplyChange(&fakeChange{})
	}()
	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Fatal("Panic expected")
			}
		}()
		store.ApplyChange(nil)
	}()
}

func TestContestStore_Create(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewContestStore(db, "test_contest", "test_contest_change")
	for i := 0; i < 10; i++ {
		if err := store.Create(
			&Contest{0, 0, time.Now().Unix()},
		); err != nil {
			t.Fatal(err)
		}
	}
}

func TestContestStore_Modify(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewContestStore(db, "test_contest", "test_contest_change")
	contest := Contest{
		CreateTime: 1,
	}
	if err := store.Create(&contest); err != nil {
		t.Fatal(err)
	}
	if contest.ID <= 0 {
		t.Fatal("ID should be greater that zero")
	}
	found, ok := store.Get(contest.ID)
	if !ok {
		t.Fatal("Unable to found contest")
	}
	if found.CreateTime != contest.CreateTime {
		t.Fatal("Contest has invalid create time")
	}
	contest.CreateTime = 2
	if err := store.Update(&contest); err != nil {
		t.Fatal(err)
	}
	found, ok = store.Get(contest.ID)
	if !ok {
		t.Fatal("Unable to found contest")
	}
	if found.CreateTime != contest.CreateTime {
		t.Fatal("Contest has invalid create time")
	}
	if err := store.Delete(contest.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Get(contest.ID); ok {
		t.Fatal("Contest should be deleted")
	}
}

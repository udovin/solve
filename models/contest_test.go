package models

import (
	"testing"
	"time"
)

func TestContestStore_getLocker(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewContestStore(db, "test_contest", "test_contest_change")
	if store.getLocker() == nil {
		t.Fatal("locker should not be nil")
	}
}

func TestContestStore_applyChange(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewContestStore(db, "test_contest", "test_contest_change")
	store.applyChange(&contestChange{
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
	store.applyChange(&contestChange{
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
	store.applyChange(&contestChange{
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
		store.applyChange(&contestChange{
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
		store.applyChange(&fakeChange{})
	}()
	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Fatal("Panic expected")
			}
		}()
		store.applyChange(nil)
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

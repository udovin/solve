package models

import (
	"testing"
)

func TestContestStore_GetDB(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewContestStore(db, "test_contest", "test_contest_change")
	if store.GetDB() != db {
		t.Error("Store has invalid database")
	}
}

func TestContestStore_applyChange(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewContestStore(db, "test_contest", "test_contest_change")
	store.applyChange(&ContestChange{
		BaseChange: BaseChange{ID: 1, Type: CreateChange, Time: 0},
		Contest:    Contest{ID: 1, OwnerID: 1},
	})
	m, ok := store.Get(1)
	if !ok {
		t.Error("Contest should exists")
	}
	if m.OwnerID != 1 {
		t.Error("Wrong owner ID")
	}
	store.applyChange(&ContestChange{
		BaseChange: BaseChange{ID: 2, Type: UpdateChange, Time: 1},
		Contest:    Contest{ID: 1, OwnerID: 2},
	})
	m, ok = store.Get(1)
	if !ok {
		t.Error("Contest should exists")
	}
	if m.OwnerID != 2 {
		t.Error("Wrong owner ID")
	}
	store.applyChange(&ContestChange{
		BaseChange: BaseChange{ID: 3, Type: DeleteChange, Time: 2},
		Contest:    Contest{ID: 1},
	})
	if _, ok := store.Get(1); ok {
		t.Error("Contest should be deleted")
	}
	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Error("Panic expected")
			}
		}()
		store.applyChange(&ContestChange{
			BaseChange: BaseChange{ID: 4, Type: ChangeType(126), Time: 0},
			Contest:    Contest{ID: 2, OwnerID: 1},
		})
	}()
	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Error("Panic expected")
			}
		}()
		store.applyChange(&MockChange{})
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

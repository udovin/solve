package models

import (
	"testing"
)

func TestPermissionStore_GetDB(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewPermissionStore(db, "test_permission", "test_permission_change")
	if store.GetDB() != db {
		t.Error("Store has invalid database")
	}
}

func TestPermissionStore_applyChange(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewPermissionStore(
		db, "test_permission", "test_permission_change",
	)
	store.applyChange(&PermissionChange{
		BaseChange: BaseChange{ID: 1, Type: CreateChange, Time: 0},
		Permission: Permission{ID: 1, Code: "test"},
	})
	m, ok := store.Get(1)
	if !ok {
		t.Error("Permission should exists")
	}
	if m.Code != "test" {
		t.Error("Wrong permission code")
	}
	store.applyChange(&PermissionChange{
		BaseChange: BaseChange{ID: 2, Type: UpdateChange, Time: 1},
		Permission: Permission{ID: 1, Code: "new_test"},
	})
	m, ok = store.Get(1)
	if !ok {
		t.Error("Permission should exists")
	}
	if m.Code != "new_test" {
		t.Error("Wrong permission code")
	}
	store.applyChange(&PermissionChange{
		BaseChange: BaseChange{ID: 3, Type: DeleteChange, Time: 2},
		Permission: Permission{ID: 1},
	})
	if _, ok := store.Get(1); ok {
		t.Error("Permission should be deleted")
	}
	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Error("Panic expected")
			}
		}()
		store.applyChange(&PermissionChange{
			BaseChange: BaseChange{ID: 4, Type: ChangeType(126), Time: 0},
			Permission: Permission{ID: 2, Code: "test"},
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

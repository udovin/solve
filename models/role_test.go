package models

import (
	"testing"
)

func TestRoleStore_GetDB(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewRoleStore(db, "test_role", "test_role_change")
	if store.GetDB() != db {
		t.Error("Store has invalid database")
	}
}

package models

import (
	"testing"
)

func TestRoleStore_getLocker(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewRoleStore(db, "test_role", "test_role_change")
	if store.getLocker() == nil {
		t.Error("Locker should not be nil")
	}
}

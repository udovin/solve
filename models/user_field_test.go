package models

import (
	"testing"
)

func TestUserFieldStore_getLocker(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewUserFieldStore(
		db, "test_user_field", "test_user_field_change",
	)
	if store.getLocker() == nil {
		t.Fatal("Locker should not be nil")
	}
}

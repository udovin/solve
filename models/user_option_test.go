package models

import (
	"testing"
)

func TestUserOptionStore_getLocker(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewUserOptionStore(
		db, "test_user_option", "test_user_option_change",
	)
	if store.getLocker() == nil {
		t.Error("Locker should not be nil")
	}
}

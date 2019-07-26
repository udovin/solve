package models

import (
	"testing"
)

func TestUserStore_getLocker(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewUserStore(db, "test_user", "test_user_change")
	if store.getLocker() == nil {
		t.Error("Locker should not be nil")
	}
}

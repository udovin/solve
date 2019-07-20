package models

import (
	"testing"
)

func TestUserStore_GetDB(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewUserStore(db, "test_user", "test_user_change")
	if store.GetDB() != db {
		t.Error("Store has invalid database")
	}
}

package models

import (
	"testing"
)

func TestSessionStore_GetDB(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewSessionStore(db, "test_session", "test_session_change")
	if store.GetDB() != db {
		t.Error("Store has invalid database")
	}
}

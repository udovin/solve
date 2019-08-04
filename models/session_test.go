package models

import (
	"testing"
)

func TestSessionStore_getLocker(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewSessionStore(db, "test_session", "test_session_change")
	if store.getLocker() == nil {
		t.Fatal("Locker should not be nil")
	}
}

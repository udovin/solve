package models

import (
	"testing"
)

func TestSessionStore_getLocker(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewSessionStore(db, "test_session", "test_session_change")
	if store.GetLocker() == nil {
		t.Fatal("Locker should not be nil")
	}
}

func TestSessionStore_Modify(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewSessionStore(db, "test_session", "test_session_change")
	session := Session{
		CreateTime: 1,
	}
	if err := store.Create(&session); err != nil {
		t.Fatal(err)
	}
	if session.ID <= 0 {
		t.Fatal("ID should be greater that zero")
	}
	found, ok := store.Get(session.ID)
	if !ok {
		t.Fatal("Unable to found session")
	}
	if found.CreateTime != session.CreateTime {
		t.Fatal("Session has invalid create time")
	}
	session.CreateTime = 2
	if err := store.Update(&session); err != nil {
		t.Fatal(err)
	}
	found, ok = store.Get(session.ID)
	if !ok {
		t.Fatal("Unable to found session")
	}
	if found.CreateTime != session.CreateTime {
		t.Fatal("Session has invalid create time")
	}
	if err := store.Delete(session.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Get(session.ID); ok {
		t.Fatal("Session should be deleted")
	}
}

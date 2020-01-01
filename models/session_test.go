package models

import (
	"testing"
)

func TestSessionStore_getLocker(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	store := NewSessionStore(testDB, "test_session", "test_session_change")
	if store.GetLocker() == nil {
		t.Fatal("Locker should not be nil")
	}
}

func TestSessionStore_Modify(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	store := NewSessionStore(testDB, "test_session", "test_session_change")
	session := Session{
		CreateTime: 1,
	}
	if err := store.Create(&session); err != nil {
		t.Fatal(err)
	}
	if session.ID <= 0 {
		t.Fatal("ID should be greater that zero")
	}
	found, err := store.Get(session.ID)
	if err != nil {
		t.Fatal("Unable to found session")
	}
	if found.CreateTime != session.CreateTime {
		t.Fatal("Session has invalid create time")
	}
	session.CreateTime = 2
	if err := store.Update(&session); err != nil {
		t.Fatal(err)
	}
	found, err = store.Get(session.ID)
	if err != nil {
		t.Fatal("Unable to found session")
	}
	if found.CreateTime != session.CreateTime {
		t.Fatal("Session has invalid create time")
	}
	if err := store.Delete(session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(session.ID); err == nil {
		t.Fatal("Session should be deleted")
	}
}

package models

import (
	"testing"
)

func TestParticipantStore_getLocker(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewParticipantStore(testDB, "test_participant", "test_participant_change")
	if store.GetLocker() == nil {
		t.Fatal("Locker should not be nil")
	}
}

func TestParticipantStore_Modify(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewParticipantStore(testDB, "test_participant", "test_participant_change")
	participant := Participant{
		CreateTime: 1,
	}
	if err := store.Create(&participant); err != nil {
		t.Fatal(err)
	}
	if participant.ID <= 0 {
		t.Fatal("ID should be greater that zero")
	}
	found, err := store.Get(participant.ID)
	if err != nil {
		t.Fatal("Unable to found participant")
	}
	if found.CreateTime != participant.CreateTime {
		t.Fatal("Participant has invalid create time")
	}
	participant.CreateTime = 2
	if err := store.Update(&participant); err != nil {
		t.Fatal(err)
	}
	found, err = store.Get(participant.ID)
	if err != nil {
		t.Fatal("Unable to found participant")
	}
	if found.CreateTime != participant.CreateTime {
		t.Fatal("Participant has invalid create time")
	}
	if err := store.Delete(participant.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(participant.ID); err == nil {
		t.Fatal("Participant should be deleted")
	}
}

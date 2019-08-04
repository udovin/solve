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

func TestUserFieldStore_Modify(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewUserFieldStore(
		db, "test_user_field", "test_user_field_change",
	)
	field := UserField{
		Data: "value1",
	}
	if err := store.Create(&field); err != nil {
		t.Fatal(err)
	}
	if field.ID <= 0 {
		t.Fatal("ID should be greater that zero")
	}
	found, ok := store.Get(field.ID)
	if !ok {
		t.Fatal("Unable to found field")
	}
	if found.Data != field.Data {
		t.Fatal("User field has invalid create time")
	}
	field.Data = "value2"
	if err := store.Update(&field); err != nil {
		t.Fatal(err)
	}
	found, ok = store.Get(field.ID)
	if !ok {
		t.Fatal("Unable to found field field")
	}
	if found.Data != field.Data {
		t.Fatal("User field has invalid create time")
	}
	if err := store.Delete(field.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Get(field.ID); ok {
		t.Fatal("User field should be deleted")
	}
}

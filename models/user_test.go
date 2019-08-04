package models

import (
	"testing"
)

func TestUserStore_getLocker(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewUserStore(db, "test_user", "test_user_change")
	if store.getLocker() == nil {
		t.Fatal("Locker should not be nil")
	}
}

func TestUserStore_Modify(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewUserStore(db, "test_user", "test_user_change")
	user := User{
		CreateTime: 1,
	}
	if err := store.Create(&user); err != nil {
		t.Fatal(err)
	}
	if user.ID <= 0 {
		t.Fatal("ID should be greater that zero")
	}
	found, ok := store.Get(user.ID)
	if !ok {
		t.Fatal("Unable to found user")
	}
	if found.CreateTime != user.CreateTime {
		t.Fatal("User has invalid create time")
	}
	user.CreateTime = 2
	if err := store.Update(&user); err != nil {
		t.Fatal(err)
	}
	found, ok = store.Get(user.ID)
	if !ok {
		t.Fatal("Unable to found user")
	}
	if found.CreateTime != user.CreateTime {
		t.Fatal("User has invalid create time")
	}
	if err := store.Delete(user.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Get(user.ID); ok {
		t.Fatal("User should be deleted")
	}
}

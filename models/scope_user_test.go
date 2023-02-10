package models

import (
	"testing"
)

func TestScopeUserStorePassword(t *testing.T) {
	store := NewScopeUserStore(nil, "", "", "qwerty")
	user := ScopeUser{}
	store.SetPassword(&user, "password")
	if !store.CheckPassword(user, "password") {
		t.Fatal("Expected same password")
	}
}

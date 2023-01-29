package models

import (
	"testing"
)

func TestScopeUserStorePassword(t *testing.T) {
	store := NewScopeUserStore(nil, "", "", "qwerty")
	user := ScopeUser{}
	store.SetPassword(&user, "password")
	if p, err := store.GetPassword(user); err != nil {
		t.Fatal("Error:", err)
	} else if p != "password" {
		t.Fatalf("Expected %q, got %q", "password", p)
	}
}

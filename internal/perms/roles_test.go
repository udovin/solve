package perms

import (
	"testing"
)

func TestRole_IsBuiltIn(t *testing.T) {
	for name := range builtInRoles {
		if !IsBuiltInRole(name) {
			t.Fatalf("Expected built-in role %q", name)
		}
	}
	if name := "unknown_role"; IsBuiltInRole(name) {
		t.Fatalf("Expected custom role %q", name)
	}
}

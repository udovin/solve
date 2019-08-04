package models

import (
	"testing"
)

func TestProblemStore_getLocker(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewProblemStore(db, "test_problem", "test_problem_change")
	if store.getLocker() == nil {
		t.Fatal("Locker should not be nil")
	}
}

package models

import (
	"testing"
)

func TestProblemStore_GetDB(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewProblemStore(db, "test_problem", "test_problem_change")
	if store.GetDB() != db {
		t.Error("Store has invalid database")
	}
}

package models

import (
	"testing"
)

func TestSolutionStore_getLocker(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	store := NewSolutionStore(testDB, "test_solution", "test_solution_change")
	if store.GetLocker() == nil {
		t.Fatal("Locker should not be nil")
	}
}

func TestSolutionStore_Modify(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	store := NewSolutionStore(testDB, "test_solution", "test_solution_change")
	solution := Solution{
		CreateTime: 1,
	}
	if err := store.Create(&solution); err != nil {
		t.Fatal(err)
	}
	if solution.ID <= 0 {
		t.Fatal("ID should be greater that zero")
	}
	found, err := store.Get(solution.ID)
	if err != nil {
		t.Fatal("Unable to found solution")
	}
	if found.CreateTime != solution.CreateTime {
		t.Fatal("Solution has invalid create time")
	}
	solution.CreateTime = 2
	if err := store.Update(&solution); err != nil {
		t.Fatal(err)
	}
	found, err = store.Get(solution.ID)
	if err != nil {
		t.Fatal("Unable to found solution")
	}
	if found.CreateTime != solution.CreateTime {
		t.Fatal("Solution has invalid create time")
	}
	if err := store.Delete(solution.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(solution.ID); err == nil {
		t.Fatal("Solution should be deleted")
	}
}

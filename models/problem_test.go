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

func TestProblemStore_Modify(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewProblemStore(db, "test_problem", "test_problem_change")
	problem := Problem{
		CreateTime: 1,
	}
	if err := store.Create(&problem); err != nil {
		t.Fatal(err)
	}
	if problem.ID <= 0 {
		t.Fatal("ID should be greater that zero")
	}
	found, ok := store.Get(problem.ID)
	if !ok {
		t.Fatal("Unable to found problem")
	}
	if found.CreateTime != problem.CreateTime {
		t.Fatal("Problem has invalid create time")
	}
	problem.CreateTime = 2
	if err := store.Update(&problem); err != nil {
		t.Fatal(err)
	}
	found, ok = store.Get(problem.ID)
	if !ok {
		t.Fatal("Unable to found problem")
	}
	if found.CreateTime != problem.CreateTime {
		t.Fatal("Problem has invalid create time")
	}
	if err := store.Delete(problem.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Get(problem.ID); ok {
		t.Fatal("Problem should be deleted")
	}
}

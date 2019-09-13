package models

import (
	"testing"
)

func TestContestProblemStore_getLocker(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewContestProblemStore(db, "test_contest_problem", "test_contest_problem_change")
	if store.GetLocker() == nil {
		t.Fatal("Locker should not be nil")
	}
}

func TestContestProblemStore_Modify(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewContestProblemStore(db, "test_contest_problem", "test_contest_problem_change")
	contestProblem := ContestProblem{
		ContestID: 1,
		ProblemID: 2,
		Code:      "Test",
	}
	if err := store.Create(&contestProblem); err != nil {
		t.Fatal(err)
	}
	found := store.GetByContest(contestProblem.ContestID)
	if len(found) != 1 {
		t.Fatal("Unable to found contest problem")
	}
	if found[0].Code != contestProblem.Code {
		t.Fatal("Contest problem has invalid problem ID")
	}
	contestProblem.Code = "Test2"
	if err := store.Update(&contestProblem); err != nil {
		t.Fatal(err)
	}
	found = store.GetByContest(contestProblem.ContestID)
	if len(found) != 1 {
		t.Fatal("Unable to found contest problem")
	}
	if found[0].Code != contestProblem.Code {
		t.Fatal("Contest problem has invalid problem ID")
	}
	if err := store.Delete(
		contestProblem.ContestID, contestProblem.ProblemID,
	); err != nil {
		t.Fatal(err)
	}
	if found := store.GetByContest(contestProblem.ContestID); len(found) != 0 {
		t.Fatal("ContestProblem should be deleted")
	}
}

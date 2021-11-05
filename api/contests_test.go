package api

import "testing"

var testCreateContest = createContestForm{
	Title: "Test contest",
}

func TestContestSimpleScenario(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	client := newTestClient()
	if _, err := client.Register(testRegisterUser); err != nil {
		t.Fatal("Error:", err)
	}
	testSyncManagers(t)
	if _, err := testSocketCreateUserRole("test", "observe_contest"); err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := testSocketCreateUserRole("test", "create_contest"); err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := testSocketCreateUserRole("test", "update_contest"); err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := testSocketCreateUserRole("test", "delete_contest"); err != nil {
		t.Fatal("Error:", err)
	}
	testSyncManagers(t)
	if _, err := client.Login("test", "qwerty123"); err != nil {
		t.Fatal("Error:", err)
	}
	contests, err := client.ObserveContests()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if len(contests.Contests) > 0 {
		t.Fatal("Contests list should be empty")
	}
	contest, err := client.CreateContest(testCreateContest)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if contest.ID == 0 {
		t.Fatal("Invalid contest ID")
	}
	if contest.Title != "Test contest" {
		t.Fatal("Invalid title:", contest.Title)
	}
}

package api

import (
	"fmt"
	"math/rand"
	"testing"
)

var testContestTitle = "Test contest"

var testCreateContest = createContestForm{
	Title: &testContestTitle,
}

func TestContestSimpleScenario(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	client := newTestClient()
	if _, err := client.Register(testRegisterUser); err != nil {
		t.Fatal("Error:", err)
	}
	testSyncManagers(t)
	if err := testSocketCreateUserRoles(
		"test", "observe_contest", "create_contest", "update_contest", "delete_contest",
	); err != nil {
		t.Fatal("Error:", err)
	}
	testSyncManagers(t)
	if _, err := client.Login("test", "qwerty123"); err != nil {
		t.Fatal("Error:", err)
	}
	{
		contests, err := client.ObserveContests()
		if err != nil {
			t.Fatal("Error:", err)
		}
		if len(contests.Contests) != 0 {
			t.Fatal("Contests list should be empty")
		}
	}
	contest, err := client.CreateContest(testCreateContest)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if contest.ID == 0 {
		t.Fatal("Invalid contest ID")
	}
	if contest.Title != testContestTitle {
		t.Fatal("Invalid title:", contest.Title)
	}
	testSyncManagers(t)
	{
		contests, err := client.ObserveContests()
		if err != nil {
			t.Fatal("Error:", err)
		}
		if len(contests.Contests) != 1 {
			t.Fatal("Contests list should not be empty")
		}
	}
	{
		created, err := client.ObserveContest(contest.ID)
		if err != nil {
			t.Fatal("Error:", err)
		}
		if created.ID != contest.ID {
			t.Fatal("Invalid contest ID:", created.ID)
		}
		if created.Title != contest.Title {
			t.Fatal("Invalid title:", created.Title)
		}
	}
}

func ptrString(s string) *string {
	return &s
}

func BenchmarkContests(b *testing.B) {
	rnd := rand.New(rand.NewSource(42))
	testSetup(b)
	defer testTeardown(b)
	client := newTestClient()
	if _, err := client.Register(testRegisterUser); err != nil {
		b.Fatal("Error:", err)
	}
	testSyncManagers(b)
	if err := testSocketCreateUserRoles(
		"test", "observe_contest", "create_contest", "update_contest", "delete_contest",
	); err != nil {
		b.Fatal("Error:", err)
	}
	testSyncManagers(b)
	if _, err := client.Login("test", "qwerty123"); err != nil {
		b.Fatal("Error:", err)
	}
	b.ResetTimer()
	var ids []int64
	for i := 0; i < b.N; i++ {
		form := createContestForm{
			Title: ptrString(fmt.Sprintf("Contest %d", i+1)),
		}
		contest, err := client.CreateContest(form)
		if err != nil {
			b.Fatal("Error:", err)
		}
		testSyncManagers(b)
		ids = append(ids, contest.ID)
	}
	rnd.Shuffle(len(ids), func(i, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})
	for _, id := range ids {
		contest, err := client.ObserveContest(id)
		if err != nil {
			b.Fatal("Error:", err)
		}
		if contest.ID != id {
			b.Fatal("Invalid contest ID:", contest.ID)
		}
	}
}

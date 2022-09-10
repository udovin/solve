package api

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/udovin/solve/models"
)

var testSimpleContest = createContestForm{
	Title: getPtr("Test contest"),
}

var testSimpleConfiguredContest = createContestForm{
	Title:     getPtr("Test configured contest"),
	BeginTime: getPtr(NInt64(time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC).Unix())),
	Duration:  getPtr(7200),
}

func TestContestSimpleScenario(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	if _, err := testAPI.Register(testSimpleUser); err != nil {
		t.Fatal("Error:", err)
	}
	testSyncManagers(t)
	if err := testSocketCreateUserRoles(
		"test", "observe_contest", "create_contest", "update_contest", "delete_contest",
	); err != nil {
		t.Fatal("Error:", err)
	}
	testSyncManagers(t)
	if _, err := testAPI.Login(context.Background(), "test", "qwerty123"); err != nil {
		t.Fatal("Error:", err)
	}
	{
		contests, err := testAPI.ObserveContests()
		if err != nil {
			t.Fatal("Error:", err)
		}
		testCheck(contests)
	}
	contest, err := testAPI.CreateContest(testSimpleContest)
	if err != nil {
		t.Fatal("Error:", err)
	}
	testCheck(contest)
	{
		resp, err := testAPI.CreateContest(testSimpleConfiguredContest)
		if err != nil {
			t.Fatal("Error:", err)
		}
		testCheck(resp)
	}
	testSyncManagers(t)
	{
		contests, err := testAPI.ObserveContests()
		if err != nil {
			t.Fatal("Error:", err)
		}
		testCheck(contests)
	}
	{
		created, err := testAPI.ObserveContest(contest.ID)
		if err != nil {
			t.Fatal("Error:", err)
		}
		testCheck(created)
	}
	for i := 0; i < 3; i++ {
		c := testView.core
		problem := models.Problem{
			Title: fmt.Sprintf("Test problem %d", i+1),
		}
		err := c.Problems.Create(context.Background(), &problem)
		if err != nil {
			t.Fatal("Error:", err)
		}
		testSyncManagers(t)
		form := createContestProblemForm{
			Code:      fmt.Sprintf("%c", 'A'+i),
			ProblemID: problem.ID,
		}
		contestProblem, err := testAPI.CreateContestProblem(contest.ID, form)
		if err != nil {
			t.Fatal("Error:", err)
		}
		testCheck(contestProblem)
	}
}

func BenchmarkContests(b *testing.B) {
	rnd := rand.New(rand.NewSource(42))
	testSetup(b)
	defer testTeardown(b)
	if _, err := testAPI.Register(testSimpleUser); err != nil {
		b.Fatal("Error:", err)
	}
	testSyncManagers(b)
	if err := testSocketCreateUserRoles(
		"test", "observe_contest", "create_contest", "update_contest", "delete_contest",
	); err != nil {
		b.Fatal("Error:", err)
	}
	testSyncManagers(b)
	if _, err := testAPI.Login(context.Background(), "test", "qwerty123"); err != nil {
		b.Fatal("Error:", err)
	}
	b.ResetTimer()
	var ids []int64
	for i := 0; i < b.N; i++ {
		form := createContestForm{
			Title: getPtr(fmt.Sprintf("Contest %d", i+1)),
		}
		contest, err := testAPI.CreateContest(form)
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
		contest, err := testAPI.ObserveContest(id)
		if err != nil {
			b.Fatal("Error:", err)
		}
		if contest.ID != id {
			b.Fatal("Invalid contest ID:", contest.ID)
		}
	}
}

package api

import (
	"context"
	"fmt"
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
	e := NewTestEnv(t)
	defer e.Close()
	user := NewTestUser(e)
	user.AddRoles("observe_contest", "create_contest", "update_contest", "delete_contest")
	user.LoginClient()
	{
		contests, err := e.Client.ObserveContests()
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(contests)
	}
	contest, err := e.Client.CreateContest(testSimpleContest)
	if err != nil {
		t.Fatal("Error:", err)
	}
	e.Check(contest)
	{
		resp, err := e.Client.CreateContest(testSimpleConfiguredContest)
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(resp)
	}
	e.SyncStores()
	{
		contests, err := e.Client.ObserveContests()
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(contests)
	}
	{
		created, err := e.Client.ObserveContest(contest.ID)
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(created)
	}
	var fakeFile models.File
	{
		err := e.Core.Files.Create(context.Background(), &fakeFile)
		if err != nil {
			t.Fatal("Error:", err)
		}
	}
	for i := 0; i < 3; i++ {
		problem := models.Problem{
			Title:     fmt.Sprintf("Test problem %d", i+1),
			PackageID: fakeFile.ID,
		}
		err := e.Core.Problems.Create(context.Background(), &problem)
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.SyncStores()
		form := createContestProblemForm{
			Code:      fmt.Sprintf("%c", 'A'+i),
			ProblemID: problem.ID,
		}
		contestProblem, err := e.Client.CreateContestProblem(contest.ID, form)
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(contestProblem)
	}
}

func BenchmarkContests(b *testing.B) {
	e := NewTestEnv(b)
	defer e.Close()
	user := NewTestUser(e)
	user.AddRoles("observe_contest", "create_contest", "update_contest", "delete_contest")
	user.LoginClient()
	b.ResetTimer()
	var ids []int64
	for i := 0; i < b.N; i++ {
		form := createContestForm{
			Title: getPtr(fmt.Sprintf("Contest %d", i+1)),
		}
		contest, err := e.Client.CreateContest(form)
		if err != nil {
			b.Fatal("Error:", err)
		}
		e.SyncStores()
		ids = append(ids, contest.ID)
	}
	e.Rand.Shuffle(len(ids), func(i, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})
	for _, id := range ids {
		contest, err := e.Client.ObserveContest(id)
		if err != nil {
			b.Fatal("Error:", err)
		}
		if contest.ID != id {
			b.Fatal("Invalid contest ID:", contest.ID)
		}
	}
}

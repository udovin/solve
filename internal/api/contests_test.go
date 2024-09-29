package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/udovin/solve/internal/models"
)

var testSimpleContest = createContestForm{
	Title: getPtr("Test contest"),
}

var testSimpleConfiguredContest = createContestForm{
	Title:               getPtr("Test configured contest"),
	BeginTime:           getPtr(NInt64(time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC).Unix())),
	Duration:            getPtr(7200),
	EnableRegistration:  getPtr(true),
	EnableUpsolving:     getPtr(true),
	FreezeBeginDuration: getPtr(3600),
	FreezeEndTime:       getPtr(NInt64(time.Date(2020, 1, 2, 1, 0, 0, 0, time.UTC).Unix())),
	StandingsKind:       getPtr(models.ICPCStandings),
}

func TestContestSimpleScenario(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	user := NewTestUser(e)
	user.AddRoles("observe_contest", "create_contest", "update_contest", "delete_contest")
	user.LoginClient()
	defer user.LogoutClient()
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
	{
		contests, err := e.Client.ObserveContests()
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(contests)
	}
	{
		created, err := e.Client.ObserveContest(context.Background(), contest.ID)
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
			PackageID: NInt64(fakeFile.ID),
		}
		err := e.Core.Problems.Create(context.Background(), &problem)
		if err != nil {
			t.Fatal("Error:", err)
		}
		form := createContestProblemForm{
			Code:      getPtr(fmt.Sprintf("%c", 'A'+i)),
			ProblemID: getPtr(problem.ID),
		}
		contestProblem, err := e.Client.CreateContestProblem(contest.ID, form)
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(contestProblem)
	}
}

func TestContestParticipation(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	var contest Contest
	var contestProblem ContestProblem
	func() {
		user := NewTestUser(e)
		user.AddRoles("observe_contest", "create_contest", "update_contest", "delete_contest")
		user.LoginClient()
		defer user.LogoutClient()
		contestForm := createContestForm{
			Title:              getPtr("Test contest"),
			BeginTime:          getPtr(NInt64(e.Now.Add(time.Hour).Unix())),
			Duration:           getPtr(7200),
			EnableRegistration: getPtr(true),
			EnableUpsolving:    getPtr(true),
			StandingsKind:      getPtr(models.ICPCStandings),
		}
		var err error
		if contest, err = e.Client.CreateContest(contestForm); err != nil {
			t.Fatal("Error:", err)
		}
		var fakeFile models.File
		{
			err := e.Core.Files.Create(context.Background(), &fakeFile)
			if err != nil {
				t.Fatal("Error:", err)
			}
		}
		problem := models.Problem{
			Title:     "Test problem",
			PackageID: NInt64(fakeFile.ID),
		}
		if err := e.Core.Problems.Create(context.Background(), &problem); err != nil {
			t.Fatal("Error:", err)
		}
		problemForm := createContestProblemForm{
			Code:      getPtr("A"),
			ProblemID: getPtr(problem.ID),
		}
		if contestProblem, err = e.Client.CreateContestProblem(contest.ID, problemForm); err != nil {
			t.Fatal("Error:", err)
		}
	}()
	_ = contestProblem
	func() {
		user := NewTestUser(e)
		user.LoginClient()
		defer user.LogoutClient()
		now := e.Now
		if resp, err := e.Client.ObserveContest(context.Background(), contest.ID); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(resp)
		}
		e.Now = now.Add(time.Hour)
		if _, err := e.Client.ObserveContest(context.Background(), contest.ID); err == nil {
			t.Fatal("Expected error")
		} else if resp, ok := err.(statusCodeResponse); !ok {
			t.Fatal("Invalid error:", err)
		} else {
			expectStatus(t, http.StatusForbidden, resp.StatusCode())
		}
		e.Now = now.Add(3*time.Hour - 1)
		if _, err := e.Client.ObserveContest(context.Background(), contest.ID); err == nil {
			t.Fatal("Expected error")
		} else if resp, ok := err.(statusCodeResponse); !ok {
			t.Fatal("Invalid error:", err)
		} else {
			expectStatus(t, http.StatusForbidden, resp.StatusCode())
		}
		e.Now = now.Add(3*time.Hour + time.Second)
		if resp, err := e.Client.ObserveContest(context.Background(), contest.ID); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(resp)
		}
	}()
}

func TestContestStandings(t *testing.T) {
	e := NewTestEnv(t, WithInvoker{})
	defer e.Close()
	owner := NewTestUser(e)
	owner.AddRoles("create_contest", "create_compiler", "create_problem", "create_setting")
	user1 := NewTestUser(e)
	user2 := NewTestUser(e)
	user3 := NewTestUser(e)
	owner.LoginClient()
	compiler := NewTestCompiler(e)
	problem := NewTestProblem(e)
	interactiveProblem := NewTestInteractiveProblem(e)
	contestForm := createContestForm{
		Title:              getPtr("Test contest"),
		BeginTime:          getPtr(NInt64(e.Now.Add(time.Hour).Unix())),
		Duration:           getPtr(7200),
		EnableRegistration: getPtr(true),
		EnableUpsolving:    getPtr(true),
		StandingsKind:      getPtr(models.ICPCStandings),
	}
	contest, err := e.Client.CreateContest(contestForm)
	if err != nil {
		t.Fatal("Error:", err)
	}
	{
		problemForm := createContestProblemForm{}
		problemForm.Code = getPtr("A")
		problemForm.ProblemID = getPtr(problem.ID)
		if _, err := e.Client.CreateContestProblem(contest.ID, problemForm); err != nil {
			t.Fatal("Error", err)
		}
	}
	{
		problemForm := createContestProblemForm{}
		problemForm.Code = getPtr("B")
		problemForm.ProblemID = getPtr(interactiveProblem.ID)
		if _, err := e.Client.CreateContestProblem(contest.ID, problemForm); err != nil {
			t.Fatal("Error", err)
		}
	}
	{
		form := CreateContestParticipantForm{
			AccountID: user1.ID,
			Kind:      models.RegularParticipant,
		}
		if _, err := e.Client.CreateContestParticipant(context.Background(), contest.ID, form); err != nil {
			t.Fatal("Error", err)
		}
	}
	{
		form := CreateContestParticipantForm{
			AccountID: user2.ID,
			Kind:      models.RegularParticipant,
		}
		if _, err := e.Client.CreateContestParticipant(context.Background(), contest.ID, form); err != nil {
			t.Fatal("Error", err)
		}
	}
	{
		form := CreateContestParticipantForm{
			AccountID: user3.ID,
			Kind:      models.RegularParticipant,
		}
		if _, err := e.Client.CreateContestParticipant(context.Background(), contest.ID, form); err != nil {
			t.Fatal("Error", err)
		}
	}
	owner.LogoutClient()
	e.SyncStores()
	now := e.Now
	{
		user1.LoginClient()
		e.Now = now.Add(time.Hour).Add(time.Minute * 5)
		form := SubmitSolutionForm{
			CompilerID: compiler.ID,
			Content:    getPtr("CompilationError"),
		}
		solution, err := e.Client.SubmitContestSolution(context.Background(), contest.ID, "A", form)
		if err != nil {
			t.Fatal("Error", err)
		}
		// TODO: Fix submit solution handler.
		if err := e.Core.ContestSolutions.Sync(context.Background()); err != nil {
			t.Fatal("Error", err)
		}
		solutionModel, err := e.Core.ContestSolutions.Get(context.Background(), solution.ID)
		if err != nil {
			t.Fatal("Error", err)
		}
		e.WaitSolutionJudged(solutionModel.ID)
		solution, err = e.Client.ObserveContestSolution(context.Background(), contest.ID, solution.ID)
		if err != nil {
			t.Fatal("Error", err)
		}
		if v := solution.Solution.Report.Verdict; v != models.CompilationError.String() {
			t.Fatalf("Expected: %q, got: %q", models.CompilationError, solution.Solution.Report.Verdict)
		}
		if standings, err := e.Client.ObserveContestStandings(context.Background(), contest.ID); err != nil {
			t.Fatal("Error:", err)
		} else {
			e.Check(standings)
		}
		user1.LogoutClient()
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
		ids = append(ids, contest.ID)
	}
	e.Rand.Shuffle(len(ids), func(i, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})
	for _, id := range ids {
		contest, err := e.Client.ObserveContest(context.Background(), id)
		if err != nil {
			b.Fatal("Error:", err)
		}
		if contest.ID != id {
			b.Fatal("Invalid contest ID:", contest.ID)
		}
	}
}

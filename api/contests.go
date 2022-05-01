package api

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

func (v *View) registerContestHandlers(g *echo.Group) {
	g.GET(
		"/v0/contests", v.observeContests,
		v.sessionAuth,
		v.requireAuthRole(models.ObserveContestsRole),
	)
	g.POST(
		"/v0/contests", v.createContest,
		v.sessionAuth, v.requireAuth,
		v.requireAuthRole(models.CreateContestRole),
	)
	g.GET(
		"/v0/contests/:contest", v.observeContest,
		v.sessionAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.ObserveContestRole),
	)
	g.PATCH(
		"/v0/contests/:contest", v.updateContest,
		v.sessionAuth, v.requireAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.UpdateContestRole),
	)
	g.DELETE(
		"/v0/contests/:contest", v.deleteContest,
		v.sessionAuth, v.requireAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.DeleteContestRole),
	)
	g.GET(
		"/v0/contests/:contest/problems", v.observeContestProblems,
		v.sessionAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.ObserveContestProblemsRole),
	)
	g.GET(
		"/v0/contests/:contest/problems/:problem", v.observeContestProblem,
		v.sessionAuth, v.extractContest, v.extractContestProblem,
		v.extractContestRoles,
		v.requireAuthRole(models.ObserveContestProblemRole),
	)
	g.POST(
		"/v0/contests/:contest/problems", v.createContestProblem,
		v.sessionAuth, v.requireAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.CreateContestProblemRole),
	)
	g.DELETE(
		"/v0/contests/:contest/problems/:problem", v.deleteContestProblem,
		v.sessionAuth, v.requireAuth, v.extractContest,
		v.extractContestProblem, v.extractContestRoles,
		v.requireAuthRole(models.DeleteContestProblemRole),
	)
	g.POST(
		"/v0/contests/:contest/problems/:problem/submit",
		v.submitContestProblemSolution, v.sessionAuth, v.requireAuth,
		v.extractContest, v.extractContestProblem, v.extractContestRoles,
		v.requireAuthRole(models.SubmitContestSolutionRole),
	)
	g.GET(
		"/v0/contests/:contest/solutions", v.observeContestSolutions,
		v.sessionAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.ObserveContestSolutionsRole),
	)
	g.GET(
		"/v0/contests/:contest/solutions/:solution", v.observeContestSolution,
		v.sessionAuth, v.extractContest, v.extractContestSolution,
		v.extractContestSolutionRoles,
		v.requireAuthRole(models.ObserveContestSolutionRole),
	)
	g.GET(
		"/v0/contests/:contest/participants", v.observeContestParticipants,
		v.sessionAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.ObserveContestParticipantsRole),
	)
	g.POST(
		"/v0/contests/:contest/participants", v.createContestParticipant,
		v.sessionAuth, v.requireAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.CreateContestParticipantRole),
	)
	g.DELETE(
		"/v0/contests/:contest/participants/:participant",
		v.deleteContestParticipant, v.sessionAuth, v.requireAuth,
		v.extractContest, v.extractContestParticipant, v.extractContestRoles,
		v.requireAuthRole(models.DeleteContestParticipantRole),
	)
}

type Contest struct {
	ID          int64    `json:"id"`
	Title       string   `json:"title"`
	Permissions []string `json:"permissions,omitempty"`
}

type Contests struct {
	Contests []Contest `json:"contests"`
}

type ContestProblem struct {
	Problem
	ContestID int64  `json:"contest_id"`
	Code      string `json:"code"`
}

type ContestProblems struct {
	Problems []ContestProblem `json:"problems"`
}

type contestSorter []Contest

func (v contestSorter) Len() int {
	return len(v)
}

func (v contestSorter) Less(i, j int) bool {
	return v[i].ID > v[j].ID
}

func (v contestSorter) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

type contestSolutionSorter []ContestSolution

func (v contestSolutionSorter) Len() int {
	return len(v)
}

func (v contestSolutionSorter) Less(i, j int) bool {
	return v[i].ID > v[j].ID
}

func (v contestSolutionSorter) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

var contestPermissions = []string{
	models.UpdateContestRole,
	models.DeleteContestRole,
	models.ObserveContestProblemsRole,
	models.CreateContestProblemRole,
	models.DeleteContestProblemRole,
	models.ObserveContestParticipantsRole,
	models.CreateContestParticipantRole,
	models.DeleteContestParticipantRole,
	models.ObserveContestSolutionsRole,
	models.CreateContestSolutionRole,
	models.SubmitContestSolutionRole,
	models.UpdateContestSolutionRole,
	models.DeleteContestSolutionRole,
}

func makeContest(contest models.Contest, roles core.RoleSet, core *core.Core) Contest {
	resp := Contest{ID: contest.ID, Title: contest.Title}
	if roles != nil {
		for _, permission := range contestPermissions {
			if ok, err := core.HasRole(roles, permission); err == nil && ok {
				resp.Permissions = append(resp.Permissions, permission)
			}
		}
	}
	return resp
}

func makeContestProblem(
	contestProblem models.ContestProblem, problems *models.ProblemStore,
) ContestProblem {
	resp := ContestProblem{
		ContestID: contestProblem.ContestID,
		Code:      contestProblem.Code,
	}
	if problem, err := problems.Get(contestProblem.ProblemID); err == nil {
		resp.Problem = makeProblem(problem)
	}
	return resp
}

func (v *View) observeContests(c echo.Context) error {
	roles, ok := c.Get(authRolesKey).(core.RoleSet)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	var resp Contests
	contests, err := v.core.Contests.All()
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	for _, contest := range contests {
		contestRoles := v.extendContestRoles(c, roles, contest)
		if ok, err := v.core.HasRole(contestRoles, models.ObserveContestRole); ok && err == nil {
			resp.Contests = append(resp.Contests, makeContest(contest, contestRoles, v.core))
		}
	}
	sort.Sort(contestSorter(resp.Contests))
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeContest(c echo.Context) error {
	contest, ok := c.Get(contestKey).(models.Contest)
	if !ok {
		c.Logger().Error("contest not extracted")
		return fmt.Errorf("contest not extracted")
	}
	roles, ok := c.Get(authRolesKey).(core.RoleSet)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	return c.JSON(http.StatusOK, makeContest(contest, roles, v.core))
}

type updateContestForm struct {
	Title *string `json:"title"`
}

func (f updateContestForm) validate() *errorResponse {
	errors := errorFields{}
	if f.Title != nil {
		if len(*f.Title) < 4 {
			errors["title"] = errorField{Message: "title is too short"}
		} else if len(*f.Title) > 64 {
			errors["title"] = errorField{Message: "title is too long"}
		}
	}
	if len(errors) > 0 {
		return &errorResponse{
			Message:       "form has invalid fields",
			InvalidFields: errors,
		}
	}
	return nil
}

func (f updateContestForm) Update(contest *models.Contest) *errorResponse {
	if err := f.validate(); err != nil {
		return err
	}
	if f.Title != nil {
		contest.Title = *f.Title
	}
	return nil
}

type createContestForm updateContestForm

func (f createContestForm) Update(contest *models.Contest) *errorResponse {
	if f.Title == nil {
		return &errorResponse{
			Message: "form has invalid fields",
			InvalidFields: errorFields{
				"title": errorField{Message: "title is required"},
			},
		}
	}
	return updateContestForm(f).Update(contest)
}

func (v *View) createContest(c echo.Context) error {
	var form createContestForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var contest models.Contest
	if err := form.Update(&contest); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	if account, ok := c.Get(authAccountKey).(models.Account); ok {
		contest.OwnerID = models.NInt64(account.ID)
	}
	if err := v.core.Contests.Create(c.Request().Context(), &contest); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusCreated, makeContest(contest, nil, nil))
}

func (v *View) updateContest(c echo.Context) error {
	contest, ok := c.Get(contestKey).(models.Contest)
	if !ok {
		c.Logger().Error("contest not extracted")
		return fmt.Errorf("contest not extracted")
	}
	roles, ok := c.Get(authRolesKey).(core.RoleSet)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	var form updateContestForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if err := form.Update(&contest); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	if err := v.core.Contests.Update(c.Request().Context(), contest); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusCreated, makeContest(contest, roles, v.core))
}

func (v *View) deleteContest(c echo.Context) error {
	contest, ok := c.Get(contestKey).(models.Contest)
	if !ok {
		c.Logger().Error("contest not extracted")
		return fmt.Errorf("contest not extracted")
	}
	if err := v.core.Contests.Delete(c.Request().Context(), contest.ID); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, makeContest(contest, nil, nil))
}

func (v *View) observeContestProblems(c echo.Context) error {
	contest, ok := c.Get(contestKey).(models.Contest)
	if !ok {
		c.Logger().Error("contest not extracted")
		return fmt.Errorf("contest not extracted")
	}
	problems, err := v.core.ContestProblems.FindByContest(contest.ID)
	if err != nil {
		return err
	}
	resp := ContestProblems{}
	for _, problem := range problems {
		resp.Problems = append(
			resp.Problems,
			makeContestProblem(problem, v.core.Problems),
		)
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeContestProblem(c echo.Context) error {
	problem, ok := c.Get(contestProblemKey).(models.ContestProblem)
	if !ok {
		c.Logger().Error("contest problem not extracted")
		return fmt.Errorf("contest problem not extracted")
	}
	return c.JSON(
		http.StatusOK,
		makeContestProblem(problem, v.core.Problems),
	)
}

type createContestProblemForm struct {
	Code      string `json:"code"`
	ProblemID int64  `json:"problem_id"`
}

func (f createContestProblemForm) validate() *errorResponse {
	errors := errorFields{}
	if len(f.Code) == 0 {
		errors["code"] = errorField{Message: "code is empty"}
	}
	if len(f.Code) > 4 {
		errors["code"] = errorField{Message: "code is too long"}
	}
	if len(errors) > 0 {
		return &errorResponse{
			Message:       "form has invalid fields",
			InvalidFields: errors,
		}
	}
	return nil
}

func (f createContestProblemForm) Update(
	problem *models.ContestProblem, problems *models.ProblemStore,
) *errorResponse {
	if err := f.validate(); err != nil {
		return err
	}
	if _, err := problems.Get(f.ProblemID); err != nil {
		return &errorResponse{Message: fmt.Sprintf(
			"problem %d does not exists", f.ProblemID,
		)}
	}
	problem.Code = f.Code
	problem.ProblemID = f.ProblemID
	return nil
}

func (v *View) createContestProblem(c echo.Context) error {
	contest, ok := c.Get(contestKey).(models.Contest)
	if !ok {
		c.Logger().Error("contest not extracted")
		return fmt.Errorf("contest not extracted")
	}
	var form createContestProblemForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var problem models.ContestProblem
	if err := form.Update(&problem, v.core.Problems); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	problem.ContestID = contest.ID
	{
		problems, err := v.core.ContestProblems.FindByContest(contest.ID)
		if err != nil {
			return err
		}
		for _, contestProblem := range problems {
			if problem.Code == contestProblem.Code {
				resp := errorResponse{Message: fmt.Sprintf(
					"problem with code %q already exists", problem.Code,
				)}
				return c.JSON(http.StatusBadRequest, resp)
			}
			if problem.ProblemID == contestProblem.ProblemID {
				resp := errorResponse{Message: fmt.Sprintf(
					"problem %d already exists", problem.ProblemID,
				)}
				return c.JSON(http.StatusBadRequest, resp)
			}
		}
	}
	if err := v.core.ContestProblems.Create(c.Request().Context(), &problem); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(
		http.StatusCreated,
		makeContestProblem(problem, v.core.Problems),
	)
}

func (v *View) deleteContestProblem(c echo.Context) error {
	problem, ok := c.Get(contestProblemKey).(models.ContestProblem)
	if !ok {
		c.Logger().Error("contest problem not extracted")
		return fmt.Errorf("contest problem not extracted")
	}
	if err := v.core.ContestProblems.Delete(c.Request().Context(), problem.ID); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(
		http.StatusOK,
		makeContestProblem(problem, v.core.Problems),
	)
}

type ContestParticipant struct {
	ID        int64 `json:"id"`
	User      *User `json:"user"`
	ContestID int64 `json:"contest_id"`
	// Kind contains kind.
	Kind models.ParticipantKind `json:"kind"`
}

type ContestParticipants struct {
	Participants []ContestParticipant `json:"participants"`
}

func (v *View) observeContestParticipants(c echo.Context) error {
	contest, ok := c.Get(contestKey).(models.Contest)
	if !ok {
		c.Logger().Error("contest not extracted")
		return fmt.Errorf("contest not extracted")
	}
	participants, err := v.core.ContestParticipants.FindByContest(contest.ID)
	if err != nil {
		return err
	}
	var resp ContestParticipants
	for _, participant := range participants {
		resp.Participants = append(
			resp.Participants,
			makeContestParticipant(c, participant, v.core),
		)
	}
	return c.JSON(http.StatusOK, resp)
}

type createContestParticipantForm struct {
	UserID    *int64                 `json:"user_id"`
	UserLogin *string                `json:"user_login"`
	Kind      models.ParticipantKind `json:"kind"`
}

func (f createContestParticipantForm) Update(
	participant *models.ContestParticipant, core *core.Core,
) *errorResponse {
	if f.UserID != nil {
		user, err := core.Users.Get(*f.UserID)
		if err != nil {
			return &errorResponse{Message: fmt.Sprintf(
				"user with id %d does not exists", *f.UserID,
			)}
		}
		participant.AccountID = user.AccountID
	} else if f.UserLogin != nil {
		user, err := core.Users.GetByLogin(*f.UserLogin)
		if err != nil {
			return &errorResponse{Message: fmt.Sprintf(
				"user %q does not exists", *f.UserLogin,
			)}
		}
		participant.AccountID = user.AccountID
	}
	participant.Kind = f.Kind
	if participant.AccountID == 0 {
		return &errorResponse{
			Message: "participant account is not specified",
		}
	}
	return nil
}

func (v *View) createContestParticipant(c echo.Context) error {
	contest, ok := c.Get(contestKey).(models.Contest)
	if !ok {
		c.Logger().Error("contest not extracted")
		return fmt.Errorf("contest not extracted")
	}
	var form createContestParticipantForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var participant models.ContestParticipant
	if err := form.Update(&participant, v.core); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	participant.ContestID = contest.ID
	if err := v.core.ContestParticipants.Create(c.Request().Context(), &participant); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(
		http.StatusCreated,
		makeContestParticipant(c, participant, v.core),
	)
}

func (v *View) deleteContestParticipant(c echo.Context) error {
	participant, ok := c.Get(contestParticipantKey).(models.ContestParticipant)
	if !ok {
		c.Logger().Error("contest participant not extracted")
		return fmt.Errorf("contest participant not extracted")
	}
	if err := v.core.ContestParticipants.Delete(c.Request().Context(), participant.ID); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(
		http.StatusOK,
		makeContestParticipant(c, participant, nil),
	)
}

// ContestSolutions represents contest solutions response.
type ContestSolutions struct {
	Solutions []ContestSolution `json:"solutions"`
}

func (v *View) observeContestSolutions(c echo.Context) error {
	roles, ok := c.Get(authRolesKey).(core.RoleSet)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	contest, ok := c.Get(contestKey).(models.Contest)
	if !ok {
		c.Logger().Error("contest not extracted")
		return fmt.Errorf("contest not extracted")
	}
	solutions, err := v.core.ContestSolutions.FindByContest(contest.ID)
	if err != nil {
		return err
	}
	var resp ContestSolutions
	for _, solution := range solutions {
		solutionRoles := v.extendContestSolutionRoles(c, roles, solution)
		if ok, err := v.core.HasRole(
			solutionRoles, models.ObserveContestSolutionRole,
		); ok && err == nil {
			resp.Solutions = append(
				resp.Solutions,
				makeBaseContestSolution(c, solution, roles, v.core),
			)
		}
	}
	sort.Sort(contestSolutionSorter(resp.Solutions))
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeContestSolution(c echo.Context) error {
	roles, ok := c.Get(authRolesKey).(core.RoleSet)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	solution, ok := c.Get(contestSolutionKey).(models.ContestSolution)
	if !ok {
		c.Logger().Error("solution not extracted")
		return fmt.Errorf("solution not extracted")
	}
	resp := makeContestSolution(c, solution, roles, v.core)
	return c.JSON(http.StatusOK, resp)
}

type TestReport struct {
	Verdict  models.Verdict `json:"verdict"`
	CheckLog string         `json:"check_log,omitempty"`
	Input    string         `json:"input,omitempty"`
	Output   string         `json:"output,omitempty"`
}

type SolutionReport struct {
	Verdict    models.Verdict `json:"verdict"`
	Tests      []TestReport   `json:"tests,omitempty"`
	CompileLog string         `json:"compile_log,omitempty"`
}

type ContestSolution struct {
	ID          int64               `json:"id"`
	ContestID   int64               `json:"contest_id"`
	Problem     *ContestProblem     `json:"problem,omitempty"`
	Participant *ContestParticipant `json:"participant,omitempty"`
	Report      *SolutionReport     `json:"report"`
	CreateTime  int64               `json:"create_time"`
}

func (v *View) submitContestProblemSolution(c echo.Context) error {
	roles, ok := c.Get(authRolesKey).(core.RoleSet)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	problem, ok := c.Get(contestProblemKey).(models.ContestProblem)
	if !ok {
		return fmt.Errorf("contest problem not extracted")
	}
	contest, ok := c.Get(contestKey).(models.Contest)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	account, ok := c.Get(authAccountKey).(models.Account)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	participants, err := v.core.ContestParticipants.
		FindByContestAccount(contest.ID, account.ID)
	if err != nil {
		return err
	}
	if len(participants) == 0 {
		resp := errorResponse{Message: "participant not found"}
		return c.JSON(http.StatusForbidden, resp)
	}
	solution := models.Solution{
		ProblemID:  problem.ProblemID,
		AuthorID:   account.ID,
		CreateTime: time.Now().Unix(),
	}
	contestSolution := models.ContestSolution{
		ContestID:     contest.ID,
		ParticipantID: participants[0].ID,
		ProblemID:     problem.ID,
	}
	if err := v.core.WrapTx(c.Request().Context(), func(ctx context.Context) error {
		file, err := c.FormFile("file")
		if err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer func() {
			_ = src.Close()
		}()
		if err := v.core.Solutions.Create(ctx, &solution); err != nil {
			return err
		}
		contestSolution.SolutionID = solution.ID
		if err := v.core.ContestSolutions.Create(ctx, &contestSolution); err != nil {
			return err
		}
		task := models.Task{Kind: models.JudgeSolution}
		if err := task.SetConfig(models.JudgeSolutionConfig{
			SolutionID: solution.ID,
		}); err != nil {
			return err
		}
		if err := v.core.Tasks.Create(ctx, &task); err != nil {
			return err
		}
		dst, err := os.Create(filepath.Join(
			v.core.Config.Storage.SolutionsDir,
			fmt.Sprintf("%d.txt", solution.ID),
		))
		if err != nil {
			return err
		}
		defer dst.Close()
		_, err = io.Copy(dst, src)
		return err
	}, sqlRepeatableRead); err != nil {
		return err
	}
	return c.JSON(
		http.StatusCreated,
		makeContestSolution(c, contestSolution, roles, v.core),
	)
}

func makeBaseContestSolution(c echo.Context, solution models.ContestSolution, roles core.RoleSet, core *core.Core) ContestSolution {
	resp := ContestSolution{
		ID:        solution.ID,
		ContestID: solution.ContestID,
	}
	if baseSolution, err := core.Solutions.Get(solution.SolutionID); err == nil {
		resp.CreateTime = baseSolution.CreateTime
		resp.Report = makeBaseSolutionReport(baseSolution)
	}
	if problem, err := core.ContestProblems.Get(solution.ProblemID); err == nil {
		problemResp := makeContestProblem(problem, core.Problems)
		resp.Problem = &problemResp
	}
	if participant, err := core.ContestParticipants.Get(solution.ParticipantID); err == nil {
		participantResp := makeContestParticipant(c, participant, core)
		resp.Participant = &participantResp
	}
	return resp
}

func makeContestSolution(c echo.Context, solution models.ContestSolution, roles core.RoleSet, core *core.Core) ContestSolution {
	resp := ContestSolution{
		ID:        solution.ID,
		ContestID: solution.ContestID,
	}
	if baseSolution, err := core.Solutions.Get(solution.SolutionID); err == nil {
		resp.CreateTime = baseSolution.CreateTime
		resp.Report = makeSolutionReport(baseSolution)
	}
	if problem, err := core.ContestProblems.Get(solution.ProblemID); err == nil {
		problemResp := makeContestProblem(problem, core.Problems)
		resp.Problem = &problemResp
	}
	if participant, err := core.ContestParticipants.Get(solution.ParticipantID); err == nil {
		participantResp := makeContestParticipant(c, participant, core)
		resp.Participant = &participantResp
	}
	return resp
}

func makeContestParticipant(
	c echo.Context, participant models.ContestParticipant,
	core *core.Core,
) ContestParticipant {
	resp := ContestParticipant{
		ID:        participant.ID,
		ContestID: participant.ContestID,
		Kind:      participant.Kind,
	}
	if core != nil {
		if account, err := core.Accounts.Get(participant.AccountID); err == nil {
			switch account.Kind {
			case models.UserAccount:
				if user, err := core.Users.GetByAccount(account.ID); err == nil {
					resp.User = &User{ID: user.ID, Login: user.Login}
				}
			}
		}
	}
	return resp
}

func (v *View) extractContest(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("contest"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			resp := errorResponse{Message: "invalid contest ID"}
			return c.JSON(http.StatusBadRequest, resp)
		}
		contest, err := v.core.Contests.Get(id)
		if err == sql.ErrNoRows {
			if err := v.core.Contests.Sync(c.Request().Context()); err != nil {
				return err
			}
			contest, err = v.core.Contests.Get(id)
		}
		if err != nil {
			if err == sql.ErrNoRows {
				resp := errorResponse{Message: "contest not found"}
				return c.JSON(http.StatusNotFound, resp)
			}
			c.Logger().Error(err)
			return err
		}
		c.Set(contestKey, contest)
		return next(c)
	}
}

func (v *View) extractContestProblem(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := c.Param("problem")
		if len(code) == 0 {
			resp := errorResponse{Message: "empty problem code"}
			return c.JSON(http.StatusNotFound, resp)
		}
		contest, ok := c.Get(contestKey).(models.Contest)
		if !ok {
			c.Logger().Error("contest not extracted")
			return fmt.Errorf("contest not extracted")
		}
		problems, err := v.core.ContestProblems.FindByContest(contest.ID)
		if err != nil {
			c.Logger().Error(err)
			return err
		}
		pos := -1
		for i, problem := range problems {
			if problem.Code == code {
				pos = i
				break
			}
		}
		if pos == -1 {
			resp := errorResponse{
				Message: fmt.Sprintf("problem %q does not exists", code),
			}
			return c.JSON(http.StatusNotFound, resp)
		}
		c.Set(contestProblemKey, problems[pos])
		return next(c)
	}
}

func (v *View) extractContestParticipant(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("participant"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			resp := errorResponse{Message: "invalid participant ID"}
			return c.JSON(http.StatusBadRequest, resp)
		}
		participant, err := v.core.ContestParticipants.Get(id)
		if err != nil {
			if err == sql.ErrNoRows {
				resp := errorResponse{Message: "participant not found"}
				return c.JSON(http.StatusNotFound, resp)
			}
			c.Logger().Error(err)
			return err
		}
		if contest, ok := c.Get(contestKey).(models.Contest); ok {
			if contest.ID != participant.ContestID {
				resp := errorResponse{Message: "participant not found"}
				return c.JSON(http.StatusNotFound, resp)
			}
		}
		c.Set(contestParticipantKey, participant)
		return next(c)
	}
}

func (v *View) extractContestSolution(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("solution"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			resp := errorResponse{Message: "invalid solution ID"}
			return c.JSON(http.StatusBadRequest, resp)
		}
		solution, err := v.core.ContestSolutions.Get(id)
		if err == sql.ErrNoRows {
			if err := v.core.ContestSolutions.Sync(c.Request().Context()); err != nil {
				return err
			}
			if err := v.core.Solutions.Sync(c.Request().Context()); err != nil {
				return err
			}
			solution, err = v.core.ContestSolutions.Get(id)
		}
		if err != nil {
			if err == sql.ErrNoRows {
				resp := errorResponse{Message: "solution not found"}
				return c.JSON(http.StatusNotFound, resp)
			}
			c.Logger().Error(err)
			return err
		}
		if contest, ok := c.Get(contestKey).(models.Contest); ok {
			if contest.ID != solution.ContestID {
				resp := errorResponse{Message: "solution not found"}
				return c.JSON(http.StatusNotFound, resp)
			}
		}
		c.Set(contestSolutionKey, solution)
		return next(c)
	}
}

func (v *View) extendContestRoles(
	c echo.Context, roles core.RoleSet, contest models.Contest,
) core.RoleSet {
	contestRoles := roles.Clone()
	if contest.ID == 0 {
		return contestRoles
	}
	addRole := func(name string) {
		if err := v.core.AddRole(contestRoles, name); err != nil {
			c.Logger().Error(err)
		}
	}
	account, ok := c.Get(authAccountKey).(models.Account)
	if ok {
		if contest.OwnerID != 0 && account.ID == int64(contest.OwnerID) {
			addRole(models.ObserveContestRole)
			addRole(models.UpdateContestRole)
			addRole(models.DeleteContestRole)
			addRole(models.ObserveContestProblemsRole)
			addRole(models.ObserveContestProblemRole)
			addRole(models.CreateContestProblemRole)
			addRole(models.DeleteContestProblemRole)
			addRole(models.ObserveContestParticipantsRole)
			addRole(models.ObserveContestParticipantRole)
			addRole(models.CreateContestParticipantRole)
			addRole(models.DeleteContestParticipantRole)
			addRole(models.ObserveContestSolutionsRole)
			addRole(models.ObserveContestSolutionRole)
			addRole(models.CreateContestSolutionRole)
			addRole(models.SubmitContestSolutionRole)
			addRole(models.UpdateContestSolutionRole)
			addRole(models.DeleteContestSolutionRole)
		}
		participants, err := v.core.ContestParticipants.
			FindByContestAccount(contest.ID, account.ID)
		if err != nil {
			c.Logger().Error(err)
		} else if len(participants) > 0 {
			addRole(models.ObserveContestRole)
			// TODO(iudovin): Add support of start time.
			addRole(models.ObserveContestProblemsRole)
			addRole(models.ObserveContestProblemRole)
			addRole(models.ObserveContestSolutionsRole)
			addRole(models.SubmitContestSolutionRole)
		}
	}
	return contestRoles
}

func (v *View) extendContestSolutionRoles(
	c echo.Context, roles core.RoleSet, solution models.ContestSolution,
) core.RoleSet {
	solutionRoles := roles.Clone()
	if solution.ID == 0 {
		return solutionRoles
	}
	addRole := func(name string) {
		if err := v.core.AddRole(solutionRoles, name); err != nil {
			c.Logger().Error(err)
		}
	}
	account, ok := c.Get(authAccountKey).(models.Account)
	if ok {
		participants, err := v.core.ContestParticipants.
			FindByContestAccount(solution.ContestID, account.ID)
		if err != nil {
			c.Logger().Error(err)
		} else if len(participants) > 0 {
			for _, participant := range participants {
				if solution.ParticipantID != 0 &&
					participant.ID == solution.ParticipantID {
					addRole(models.ObserveContestSolutionRole)
				}
			}
		}
	}
	return solutionRoles
}

func (v *View) extractContestRoles(next echo.HandlerFunc) echo.HandlerFunc {
	nextWrap := func(c echo.Context) error {
		contest, ok := c.Get(contestKey).(models.Contest)
		if !ok {
			c.Logger().Error("contest not extracted")
			return fmt.Errorf("contest not extracted")
		}
		roles, ok := c.Get(authRolesKey).(core.RoleSet)
		if !ok {
			c.Logger().Error("roles not extracted")
			return fmt.Errorf("roles not extracted")
		}
		c.Set(authRolesKey, v.extendContestRoles(c, roles, contest))
		return next(c)
	}
	return v.extractAuthRoles(nextWrap)
}

func (v *View) extractContestSolutionRoles(next echo.HandlerFunc) echo.HandlerFunc {
	nextWrap := func(c echo.Context) error {
		solution, ok := c.Get(contestSolutionKey).(models.ContestSolution)
		if !ok {
			c.Logger().Error("solution not extracted")
			return fmt.Errorf("solution not extracted")
		}
		roles, ok := c.Get(authRolesKey).(core.RoleSet)
		if !ok {
			c.Logger().Error("roles not extracted")
			return fmt.Errorf("roles not extracted")
		}
		c.Set(authRolesKey, v.extendContestSolutionRoles(c, roles, solution))
		return next(c)
	}
	return v.extractContestRoles(nextWrap)
}

package api

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/internal/core"
	"github.com/udovin/solve/internal/db"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
	"github.com/udovin/solve/internal/pkg/logs"
)

func (v *View) registerContestHandlers(g *echo.Group) {
	g.GET(
		"/v0/contests", v.observeContests,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(perms.ObserveContestsRole),
	)
	g.POST(
		"/v0/contests", v.createContest,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(perms.CreateContestRole),
	)
	g.GET(
		"/v0/contests/:contest", v.observeContest,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractContest,
		v.requirePermission(perms.ObserveContestRole),
	)
	g.PATCH(
		"/v0/contests/:contest", v.updateContest,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(perms.UpdateContestRole),
	)
	g.DELETE(
		"/v0/contests/:contest", v.deleteContest,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(perms.DeleteContestRole),
	)
	g.GET(
		"/v0/contests/:contest/problems", v.observeContestProblems,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractContest,
		v.requirePermission(perms.ObserveContestProblemsRole),
	)
	g.GET(
		"/v0/contests/:contest/problems/:problem", v.observeContestProblem,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.extractContest, v.extractContestProblem,
		v.requirePermission(perms.ObserveContestProblemRole),
	)
	g.GET(
		"/v0/contests/:contest/problems/:problem/statement-files/:name",
		v.observeProblemStatementFile,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.extractContest, v.extractContestProblem,
		v.requirePermission(perms.ObserveContestProblemRole),
	)
	g.POST(
		"/v0/contests/:contest/problems", v.createContestProblem,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(perms.CreateContestProblemRole),
	)
	g.PATCH(
		"/v0/contests/:contest/problems/:problem", v.updateContestProblem,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.extractContestProblem,
		v.requirePermission(perms.UpdateContestProblemRole),
	)
	g.DELETE(
		"/v0/contests/:contest/problems/:problem", v.deleteContestProblem,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.extractContestProblem,
		v.requirePermission(perms.DeleteContestProblemRole),
	)
	g.POST(
		"/v0/contests/:contest/problems/:problem/submit",
		v.submitContestProblemSolution, v.extractAuth(v.sessionAuth),
		v.extractContest, v.extractContestProblem,
		v.requirePermission(perms.SubmitContestSolutionRole),
	)
	g.GET(
		"/v0/contests/:contest/solutions", v.observeContestSolutions,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractContest,
		v.requirePermission(perms.ObserveContestSolutionsRole),
	)
	g.GET(
		"/v0/contests/:contest/solutions/:solution", v.observeContestSolution,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.extractContest, v.extractContestSolution,
		v.requirePermission(perms.ObserveContestSolutionRole),
	)
	g.POST(
		"/v0/contests/:contest/solutions/:solution/rejudge", v.rejudgeContestSolution,
		v.extractAuth(v.sessionAuth),
		v.extractContest, v.extractContestSolution,
		v.requirePermission(perms.UpdateContestSolutionRole),
	)
	g.GET(
		"/v0/contests/:contest/participants", v.observeContestParticipants,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractContest,
		v.requirePermission(perms.ObserveContestParticipantsRole),
	)
	g.POST(
		"/v0/contests/:contest/participants", v.createContestParticipant,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(perms.CreateContestParticipantRole),
	)
	g.DELETE(
		"/v0/contests/:contest/participants/:participant",
		v.deleteContestParticipant, v.extractAuth(v.sessionAuth),
		v.extractContest, v.extractContestParticipant,
		v.requirePermission(perms.DeleteContestParticipantRole),
	)
	g.POST(
		"/v0/contests/:contest/register", v.registerContest,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(perms.RegisterContestRole),
	)
}

type ContestState struct {
	Stage string `json:"stage"`
	// Participant contains effective participant.
	Participant *ContestParticipant `json:"participant,omitempty"`
}

type Contest struct {
	ID                  int64                `json:"id"`
	Title               string               `json:"title"`
	BeginTime           NInt64               `json:"begin_time,omitempty"`
	Duration            int                  `json:"duration,omitempty"`
	Permissions         []string             `json:"permissions,omitempty"`
	EnableRegistration  bool                 `json:"enable_registration"`
	EnableUpsolving     bool                 `json:"enable_upsolving"`
	EnableObserving     bool                 `json:"enable_observing,omitempty"`
	FreezeBeginDuration int                  `json:"freeze_begin_duration,omitempty"`
	FreezeEndTime       NInt64               `json:"freeze_end_time,omitempty"`
	StandingsKind       models.StandingsKind `json:"standings_kind,omitempty"`
	State               *ContestState        `json:"state,omitempty"`
}

type Contests struct {
	Contests []Contest `json:"contests"`
}

type ContestProblem struct {
	ID        int64    `json:"id"`
	ContestID int64    `json:"contest_id"`
	Code      string   `json:"code"`
	Problem   Problem  `json:"problem"`
	Points    *int     `json:"points,omitempty"`
	Locales   []string `json:"locales,omitempty"`
	Solved    *bool    `json:"solved,omitempty"`
}

type ContestProblems struct {
	Problems []ContestProblem `json:"problems"`
}

var contestPermissions = []string{
	perms.UpdateContestRole,
	perms.UpdateContestOwnerRole,
	perms.DeleteContestRole,
	perms.RegisterContestRole,
	perms.DeregisterContestRole,
	perms.ObserveContestProblemsRole,
	perms.CreateContestProblemRole,
	perms.UpdateContestProblemRole,
	perms.DeleteContestProblemRole,
	perms.ObserveContestParticipantsRole,
	perms.CreateContestParticipantRole,
	perms.DeleteContestParticipantRole,
	perms.ObserveContestSolutionsRole,
	perms.CreateContestSolutionRole,
	perms.SubmitContestSolutionRole,
	perms.UpdateContestSolutionRole,
	perms.DeleteContestSolutionRole,
	perms.ObserveContestStandingsRole,
	perms.ObserveContestFullStandingsRole,
	perms.ObserveContestMessagesRole,
	perms.CreateContestMessageRole,
	perms.UpdateContestMessageRole,
	perms.DeleteContestMessageRole,
	perms.SubmitContestQuestionRole,
}

func makeContestStage(stage managers.ContestStage) string {
	switch stage {
	case managers.ContestNotPlanned:
		return "not_planned"
	case managers.ContestNotStarted:
		return "not_started"
	case managers.ContestStarted:
		return "started"
	case managers.ContestFinished:
		return "finished"
	default:
		return "unknown"
	}
}

func makeContest(
	c echo.Context,
	contest models.Contest,
	permissions perms.Permissions,
	core *core.Core,
) Contest {
	resp := Contest{ID: contest.ID, Title: contest.Title}
	if config, err := contest.GetConfig(); err == nil {
		resp.BeginTime = config.BeginTime
		resp.Duration = config.Duration
		resp.EnableRegistration = config.EnableRegistration
		resp.EnableUpsolving = config.EnableUpsolving
		resp.EnableObserving = config.EnableObserving
		resp.FreezeBeginDuration = config.FreezeBeginDuration
		resp.FreezeEndTime = config.FreezeEndTime
		resp.StandingsKind = config.StandingsKind
	}
	for _, permission := range contestPermissions {
		if permissions.HasPermission(permission) {
			resp.Permissions = append(resp.Permissions, permission)
		}
	}
	if contextCtx, ok := permissions.(*managers.ContestContext); ok {
		state := ContestState{
			Stage: makeContestStage(contextCtx.Stage),
		}
		participant := contextCtx.GetEffectiveParticipant()
		if core != nil && participant != nil {
			participantResp := makeContestParticipant(c, *participant, core)
			participantResp.ContestID = 0
			participantResp.User = nil
			state.Participant = getPtr(participantResp)
		}
		resp.State = &state
	}
	return resp
}

func (v *View) makeContestProblem(
	c echo.Context, contestProblem models.ContestProblem, withStatement bool,
) ContestProblem {
	resp := ContestProblem{
		ID:        contestProblem.ID,
		ContestID: contestProblem.ContestID,
		Code:      contestProblem.Code,
	}
	locales := map[string]struct{}{}
	if config, err := contestProblem.GetConfig(); err == nil {
		resp.Points = config.Points
		resp.Locales = config.Locales
		for _, locale := range config.Locales {
			locales[locale] = struct{}{}
		}
	}
	if problem, err := v.core.Problems.Get(
		getContext(c), contestProblem.ProblemID,
	); err == nil {
		resp.Problem = v.makeProblem(
			c, problem, perms.PermissionSet{}, withStatement, false, locales,
		)
		resp.Problem.Permissions = nil
	}
	return resp
}

type contestFilter struct {
	Query string `query:"q"`
}

func (f contestFilter) Filter(contest models.Contest) bool {
	if len(f.Query) > 0 {
		switch {
		case strings.HasPrefix(fmt.Sprint(contest.ID), f.Query):
		case strings.Contains(contest.Title, f.Query):
		default:
			return false
		}
	}
	return true
}

func (v *View) observeContests(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	var filter contestFilter
	if err := c.Bind(&filter); err != nil {
		c.Logger().Warn(err)
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Invalid filter."),
		}
	}
	if err := syncStore(c, v.core.Contests); err != nil {
		return err
	}
	var resp Contests
	contests, err := v.core.Contests.ReverseAll(getContext(c), 0, 0)
	if err != nil {
		return err
	}
	defer func() { _ = contests.Close() }()
	for contests.Next() {
		contest := contests.Row()
		if !filter.Filter(contest) {
			continue
		}
		contestCtx, err := v.contests.BuildContext(accountCtx, contest)
		if err != nil {
			return err
		}
		if contestCtx.HasPermission(perms.ObserveContestRole) {
			resp.Contests = append(
				resp.Contests,
				makeContest(c, contest, contestCtx, v.core),
			)
		}
	}
	if err := contests.Err(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeContest(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	return c.JSON(
		http.StatusOK,
		makeContest(c, contest, contestCtx, v.core),
	)
}

type updateContestForm struct {
	Title               *string               `json:"title" form:"title"`
	BeginTime           *NInt64               `json:"begin_time" form:"begin_time"`
	Duration            *int                  `json:"duration" form:"duration"`
	EnableRegistration  *bool                 `json:"enable_registration" form:"enable_registration"`
	EnableUpsolving     *bool                 `json:"enable_upsolving" form:"enable_upsolving"`
	EnableObserving     *bool                 `json:"enable_observing" form:"enable_observing"`
	FreezeBeginDuration *int                  `json:"freeze_begin_duration" form:"freeze_begin_duration"`
	FreezeEndTime       *NInt64               `json:"freeze_end_time" form:"freeze_end_time"`
	StandingsKind       *models.StandingsKind `json:"standings_kind" form:"standings_kind"`
	OwnerID             *int64                `json:"owner_id" form:"owner_id"`
}

func (f *updateContestForm) Update(
	c echo.Context, contest *models.Contest,
) error {
	errors := errorFields{}
	if f.Title != nil {
		if len(*f.Title) < 4 {
			errors["title"] = errorField{
				Message: localize(c, "Title is too short."),
			}
		} else if len(*f.Title) > 64 {
			errors["title"] = errorField{
				Message: localize(c, "Title is too long."),
			}
		}
		contest.Title = *f.Title
	}
	config, err := contest.GetConfig()
	if err != nil {
		return err
	}
	if f.BeginTime != nil {
		config.BeginTime = *f.BeginTime
	}
	if f.Duration != nil {
		if *f.Duration < 0 {
			errors["duration"] = errorField{
				Message: localize(c, "Duration cannot be negative."),
			}
		}
		config.Duration = *f.Duration
	}
	if f.EnableRegistration != nil {
		config.EnableRegistration = *f.EnableRegistration
	}
	if f.EnableUpsolving != nil {
		config.EnableUpsolving = *f.EnableUpsolving
	}
	if f.FreezeBeginDuration != nil {
		config.FreezeBeginDuration = *f.FreezeBeginDuration
	}
	if f.FreezeEndTime != nil {
		config.FreezeEndTime = *f.FreezeEndTime
	}
	if f.StandingsKind != nil {
		config.StandingsKind = *f.StandingsKind
	}
	if f.EnableObserving != nil {
		config.EnableObserving = *f.EnableObserving
	}
	if err := contest.SetConfig(config); err != nil {
		errors["config"] = errorField{
			Message: localize(c, "Invalid config."),
		}
	}
	if len(errors) > 0 {
		return &errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	return nil
}

type createContestForm updateContestForm

func (f *createContestForm) Update(
	c echo.Context, contest *models.Contest,
) error {
	if f.Title == nil {
		return &errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Form has invalid fields."),
			InvalidFields: errorFields{
				"title": errorField{
					Message: localize(c, "Title is required."),
				},
			},
		}
	}
	return (*updateContestForm)(f).Update(c, contest)
}

func (v *View) createContest(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	var form createContestForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var contest models.Contest
	if err := form.Update(c, &contest); err != nil {
		return err
	}
	if account := accountCtx.Account; account != nil {
		contest.OwnerID = NInt64(account.ID)
	}
	if err := v.core.Contests.Create(getContext(c), &contest); err != nil {
		return err
	}
	return c.JSON(
		http.StatusCreated,
		makeContest(c, contest, accountCtx, nil),
	)
}

func (v *View) updateContest(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	var form updateContestForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if err := form.Update(c, &contest); err != nil {
		return err
	}
	var missingPermissions []string
	if form.OwnerID != nil {
		if !contestCtx.HasPermission(perms.UpdateContestOwnerRole) {
			missingPermissions = append(missingPermissions, perms.UpdateContestOwnerRole)
		} else {
			account, err := v.core.Accounts.Get(getContext(c), *form.OwnerID)
			if err != nil {
				if err == sql.ErrNoRows {
					return errorResponse{
						Code:    http.StatusBadRequest,
						Message: localize(c, "User not found."),
					}
				}
				return err
			}
			if account.Kind != models.UserAccountKind {
				return errorResponse{
					Code:    http.StatusBadRequest,
					Message: localize(c, "User not found."),
				}
			}
			contest.OwnerID = models.NInt64(*form.OwnerID)
		}
	}
	if len(missingPermissions) > 0 {
		return errorResponse{
			Code:               http.StatusForbidden,
			Message:            localize(c, "Account missing permissions."),
			MissingPermissions: missingPermissions,
		}
	}
	if err := v.core.Contests.Update(getContext(c), contest); err != nil {
		return err
	}
	return c.JSON(
		http.StatusOK,
		makeContest(c, contest, contestCtx, v.core),
	)
}

func (v *View) deleteContest(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	if err := v.core.Contests.Delete(getContext(c), contest.ID); err != nil {
		return err
	}
	return c.JSON(
		http.StatusOK,
		makeContest(c, contest, contestCtx, nil),
	)
}

func getSolvedProblems(ctx *managers.ContestContext, c *core.Core) map[int64]bool {
	solved := map[int64]bool{}
	var participantIDs []int64
	for _, participant := range ctx.Participants {
		if participant.ID != 0 {
			participantIDs = append(participantIDs, participant.ID)
		}
	}
	solutions, err := c.ContestSolutions.FindByParticipant(ctx, participantIDs...)
	if err != nil {
		return solved
	}
	defer func() { _ = solutions.Close() }()
	for solutions.Next() {
		contestSolution := solutions.Row()
		solution, err := c.Solutions.Get(ctx, contestSolution.ID)
		if err != nil {
			continue
		}
		report, err := solution.GetReport()
		if err != nil || report == nil {
			continue
		}
		solved[contestSolution.ProblemID] = solved[contestSolution.ProblemID] ||
			report.Verdict == models.Accepted
	}
	return solved
}

func (v *View) observeContestProblems(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	solvedProblems := getSolvedProblems(contestCtx, v.core)
	problems, err := v.core.ContestProblems.FindByContest(getContext(c), contest.ID)
	if err != nil {
		return err
	}
	defer func() { _ = problems.Close() }()
	resp := ContestProblems{}
	for problems.Next() {
		problem := problems.Row()
		problemResp := v.makeContestProblem(c, problem, false)
		if v, ok := solvedProblems[problem.ID]; ok {
			problemResp.Solved = &v
		}
		resp.Problems = append(
			resp.Problems,
			problemResp,
		)
	}
	if err := problems.Err(); err != nil {
		return err
	}
	sortFunc(resp.Problems, contestProblemLess)
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeContestProblem(c echo.Context) error {
	problem, ok := c.Get(contestProblemKey).(models.ContestProblem)
	if !ok {
		return fmt.Errorf("contest problem not extracted")
	}
	return c.JSON(http.StatusOK, v.makeContestProblem(c, problem, true))
}

type updateContestProblemForm struct {
	Code      *string   `json:"code"`
	ProblemID *int64    `json:"problem_id"`
	Points    *int      `json:"points"`
	Locales   *[]string `json:"locales"`
}

func (f updateContestProblemForm) Update(
	c echo.Context,
	problem *models.ContestProblem,
	problems *models.ProblemStore,
) error {
	errors := errorFields{}
	if f.Code != nil {
		if len(*f.Code) == 0 {
			errors["code"] = errorField{
				Message: localize(c, "Code is empty."),
			}
		}
		if len(*f.Code) > 4 {
			errors["code"] = errorField{
				Message: localize(c, "Code is too long."),
			}
		}
		problem.Code = *f.Code
	}
	if len(errors) > 0 {
		return &errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	if f.ProblemID != nil {
		if _, err := problems.Get(getContext(c), *f.ProblemID); err != nil {
			return &errorResponse{
				Code: http.StatusNotFound,
				Message: localize(
					c, "Problem {id} does not exists.",
					replaceField("id", *f.ProblemID),
				),
			}
		}
		problem.ProblemID = *f.ProblemID
	}
	configUpdated := false
	config, err := problem.GetConfig()
	if err != nil {
		return err
	}
	if f.Points != nil {
		if *f.Points != 0 {
			config.Points = f.Points
		} else {
			config.Points = nil
		}
		configUpdated = true
	}
	if f.Locales != nil {
		config.Locales = *f.Locales
		configUpdated = true
	}
	if configUpdated {
		if err := problem.SetConfig(config); err != nil {
			return err
		}
	}
	return nil
}

type createContestProblemForm updateContestProblemForm

func (f *createContestProblemForm) Update(
	c echo.Context,
	problem *models.ContestProblem,
	problems *models.ProblemStore,
) error {
	if f.Code == nil {
		return &errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Form has invalid fields."),
			InvalidFields: errorFields{
				"title": errorField{
					Message: localize(c, "Code is empty."),
				},
			},
		}
	}
	if f.ProblemID == nil {
		return &errorResponse{
			Code: http.StatusNotFound,
			Message: localize(
				c, "Problem {id} does not exists.",
				replaceField("id", 0),
			),
		}
	}
	return (*updateContestProblemForm)(f).Update(c, problem, problems)
}

func (v *View) createContestProblem(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	var form createContestProblemForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if err := syncStore(c, v.core.Problems); err != nil {
		return err
	}
	var problem models.ContestProblem
	if err := form.Update(c, &problem, v.core.Problems); err != nil {
		return err
	}
	problem.ContestID = contest.ID
	if err := func() error {
		rows, err := v.core.ContestProblems.FindByContest(getContext(c), contest.ID)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			row := rows.Row()
			if problem.Code == row.Code {
				return errorResponse{
					Code: http.StatusBadRequest,
					Message: localize(
						c, "Problem with code {code} already exists.",
						replaceField("code", problem.Code),
					),
				}
			}
			if problem.ProblemID == row.ProblemID {
				return errorResponse{
					Code: http.StatusBadRequest,
					Message: localize(
						c, "Problem {id} already exists.",
						replaceField("id", problem.ProblemID),
					),
				}
			}
		}
		return rows.Err()
	}(); err != nil {
		return err
	}
	if err := v.core.ContestProblems.Create(
		getContext(c), &problem,
	); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, v.makeContestProblem(c, problem, false))
}

func (v *View) updateContestProblem(c echo.Context) error {
	problem, ok := c.Get(contestProblemKey).(models.ContestProblem)
	if !ok {
		return fmt.Errorf("contest problem not extracted")
	}
	var form updateContestProblemForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if err := form.Update(c, &problem, v.core.Problems); err != nil {
		return err
	}
	if err := v.core.ContestProblems.Update(
		getContext(c), problem,
	); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, v.makeContestProblem(c, problem, false))
}

func (v *View) deleteContestProblem(c echo.Context) error {
	problem, ok := c.Get(contestProblemKey).(models.ContestProblem)
	if !ok {
		return fmt.Errorf("contest problem not extracted")
	}
	if err := v.core.ContestProblems.Delete(
		getContext(c), problem.ID,
	); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, v.makeContestProblem(c, problem, false))
}

type ContestParticipant struct {
	ID        int64      `json:"id,omitempty"`
	User      *User      `json:"user,omitempty"`
	ScopeUser *ScopeUser `json:"scope_user,omitempty"`
	Scope     *Scope     `json:"scope,omitempty"`
	ContestID int64      `json:"contest_id,omitempty"`
	// Kind contains kind.
	Kind models.ParticipantKind `json:"kind"`
}

type ContestParticipants struct {
	Participants []ContestParticipant `json:"participants"`
}

func (v *View) observeContestParticipants(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	participants, err := v.core.ContestParticipants.FindByContest(getContext(c), contest.ID)
	if err != nil {
		return err
	}
	defer func() { _ = participants.Close() }()
	var resp ContestParticipants
	for participants.Next() {
		resp.Participants = append(
			resp.Participants,
			makeContestParticipant(c, participants.Row(), v.core),
		)
	}
	if err := participants.Err(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

type CreateContestParticipantForm struct {
	UserID      *int64                 `json:"user_id"`
	UserLogin   *string                `json:"user_login"`
	ScopeUserID *int64                 `json:"scope_user_id"`
	ScopeID     *int64                 `json:"scope_id"`
	GroupID     *int64                 `json:"group_id"`
	Kind        models.ParticipantKind `json:"kind"`
	AccountID   *int64                 `json:"account_id"`
}

func (f CreateContestParticipantForm) Update(
	c echo.Context, participant *models.ContestParticipant, core *core.Core,
) *errorResponse {
	ctx := getContext(c)
	if f.AccountID != nil {
		account, err := core.Accounts.Get(ctx, *f.AccountID)
		if err != nil {
			return &errorResponse{
				Code: http.StatusBadRequest,
				Message: localize(
					c, "Account {id} does not exists.",
					replaceField("id", *f.AccountID),
				),
			}
		}
		// TODO: Check for observe permissions.
		switch account.Kind {
		case models.UserAccountKind:
			if _, err := core.Users.Get(ctx, account.ID); err != nil {
				return &errorResponse{
					Code: http.StatusBadRequest,
					Message: localize(
						c, "User {id} does not exists.",
						replaceField("id", account.ID),
					),
				}
			}
		case models.ScopeUserAccountKind:
			scopeUser, err := core.ScopeUsers.Get(ctx, account.ID)
			if err != nil {
				return &errorResponse{
					Code: http.StatusBadRequest,
					Message: localize(
						c, "User {id} does not exists.",
						replaceField("id", account.ID),
					),
				}
			}
			if _, err := core.Scopes.Get(ctx, scopeUser.ScopeID); err != nil {
				return &errorResponse{
					Code: http.StatusBadRequest,
					Message: localize(
						c, "Scope {id} does not exists.",
						replaceField("id", scopeUser.ScopeID),
					),
				}
			}
		case models.ScopeAccountKind:
			if _, err := core.Scopes.Get(ctx, account.ID); err != nil {
				return &errorResponse{
					Code: http.StatusBadRequest,
					Message: localize(
						c, "Scope {id} does not exists.",
						replaceField("id", account.ID),
					),
				}
			}
		case models.GroupAccountKind:
			if _, err := core.Groups.Get(ctx, account.ID); err != nil {
				return &errorResponse{
					Code: http.StatusBadRequest,
					Message: localize(
						c, "Group {id} does not exists.",
						replaceField("id", *f.GroupID),
					),
				}
			}
		default:
			c.Logger().Warn(
				"Unsupported account kind",
				logs.Any("id", account.ID),
				logs.Any("kind", account.Kind),
			)
			return &errorResponse{
				Code: http.StatusBadRequest,
				Message: localize(
					c, "Account {id} does not exists.",
					replaceField("id", *f.AccountID),
				),
			}
		}
	} else if f.UserID != nil {
		user, err := core.Users.Get(ctx, *f.UserID)
		if err != nil {
			return &errorResponse{
				Code: http.StatusBadRequest,
				Message: localize(
					c, "User {id} does not exists.",
					replaceField("id", *f.UserID),
				),
			}
		}
		participant.AccountID = user.ID
	} else if f.UserLogin != nil {
		user, err := core.Users.GetByLogin(ctx, *f.UserLogin)
		if err != nil {
			return &errorResponse{
				Code: http.StatusBadRequest,
				Message: localize(
					c, "User \"{login}\" does not exists.",
					replaceField("login", *f.UserLogin),
				),
			}
		}
		participant.AccountID = user.ID
	} else if f.ScopeUserID != nil {
		user, err := core.ScopeUsers.Get(ctx, *f.ScopeUserID)
		if err != nil {
			return &errorResponse{
				Code: http.StatusBadRequest,
				Message: localize(
					c, "User {id} does not exists.",
					replaceField("id", *f.ScopeUserID),
				),
			}
		}
		participant.AccountID = user.ID
	} else if f.ScopeID != nil {
		scope, err := core.Scopes.Get(ctx, *f.ScopeID)
		if err != nil {
			return &errorResponse{
				Code: http.StatusBadRequest,
				Message: localize(
					c, "Scope {id} does not exists.",
					replaceField("id", *f.ScopeID),
				),
			}
		}
		participant.AccountID = scope.ID
	} else if f.GroupID != nil {
		group, err := core.Groups.Get(ctx, *f.GroupID)
		if err != nil {
			return &errorResponse{
				Code: http.StatusBadRequest,
				Message: localize(
					c, "Group {id} does not exists.",
					replaceField("id", *f.GroupID),
				),
			}
		}
		participant.AccountID = group.ID
	}
	participant.Kind = f.Kind
	if participant.Kind == 0 {
		participant.Kind = models.RegularParticipant
	}
	if participant.AccountID == 0 {
		return &errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Participant account is not specified."),
		}
	}
	return nil
}

func (v *View) createContestParticipant(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	var form CreateContestParticipantForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var participant models.ContestParticipant
	if err := form.Update(c, &participant, v.core); err != nil {
		return err
	}
	participant.ContestID = contest.ID
	if err := func() error {
		rows, err := v.core.ContestParticipants.FindByContestAccount(
			getContext(c), contest.ID, participant.AccountID,
		)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			row := rows.Row()
			if row.Kind == participant.Kind {
				return errorResponse{
					Code: http.StatusBadRequest,
					Message: localize(
						c, "Participant with {kind} kind already exists.",
						replaceField("kind", row.Kind),
					),
				}
			}
		}
		return rows.Err()
	}(); err != nil {
		return err
	}
	if err := v.core.ContestParticipants.Create(
		getContext(c), &participant,
	); err != nil {
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
		return fmt.Errorf("contest participant not extracted")
	}
	if err := v.core.ContestParticipants.Delete(
		getContext(c), participant.ID,
	); err != nil {
		return err
	}
	return c.JSON(
		http.StatusOK,
		makeContestParticipant(c, participant, v.core),
	)
}

func (v *View) registerContest(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	account := contestCtx.Account
	if account == nil {
		return fmt.Errorf("account not extracted")
	}
	participant := models.ContestParticipant{
		Kind:      models.RegularParticipant,
		ContestID: contest.ID,
		AccountID: account.ID,
	}
	for _, p := range contestCtx.Participants {
		if p.ID != 0 && p.Kind == participant.Kind {
			return errorResponse{
				Code: http.StatusBadRequest,
				Message: localize(
					c, "Participant with {kind} kind already exists.",
					replaceField("kind", p.Kind),
				),
			}
		}
	}
	if err := v.core.ContestParticipants.Create(
		getContext(c), &participant,
	); err != nil {
		return err
	}
	return c.JSON(
		http.StatusCreated,
		makeContestParticipant(c, participant, v.core),
	)
}

// ContestSolutions represents contest solutions response.
type ContestSolutions struct {
	Solutions   []ContestSolution `json:"solutions"`
	NextBeginID int64             `json:"next_begin_id,omitempty"`
}

type contestSolutionsFilter struct {
	ProblemID     int64          `query:"problem_id"`
	ParticipantID int64          `query:"participant_id"`
	Verdict       models.Verdict `query:"verdict"`
	BeginID       int64          `query:"begin_id"`
	Limit         int            `query:"limit"`
}

func (f *contestSolutionsFilter) Parse(c echo.Context) error {
	if err := c.Bind(f); err != nil {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Invalid filter."),
		}
	}
	if f.BeginID < 0 || f.BeginID == math.MaxInt64 {
		f.BeginID = 0
	}
	if f.Limit <= 0 {
		f.Limit = defaultSolutionLimit
	}
	f.Limit = min(f.Limit, maxSolutionLimit)
	return nil
}

func (f *contestSolutionsFilter) Filter(solution models.ContestSolution) bool {
	if f.BeginID != 0 && solution.ID > f.BeginID {
		return false
	}
	if f.ProblemID != 0 && solution.ProblemID != f.ProblemID {
		return false
	}
	if f.ParticipantID != 0 && solution.ParticipantID != f.ParticipantID {
		return false
	}
	// TODO: Filter base solution.
	return true
}

func (v *View) observeContestSolutions(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	filter := contestSolutionsFilter{Limit: 50}
	if err := filter.Parse(c); err != nil {
		c.Logger().Warn(err)
		return err
	}
	contest := contestCtx.Contest
	if err := syncStore(c, v.core.Solutions); err != nil {
		return err
	}
	if err := syncStore(c, v.core.ContestSolutions); err != nil {
		return err
	}
	var solutions db.Rows[models.ContestSolution]
	if contestCtx.HasPermission(perms.ObserveContestSolutionRole) {
		contestSolutions, err := v.core.ContestSolutions.ReverseFindByContestFrom(
			getContext(c), []int64{contest.ID}, filter.BeginID,
		)
		if err != nil {
			return err
		}
		solutions = contestSolutions
	} else {
		var participantIDs []int64
		for _, participant := range contestCtx.Participants {
			if participant.ID != 0 {
				participantIDs = append(participantIDs, participant.ID)
			}
		}
		participantSolutions, err := v.core.ContestSolutions.ReverseFindByParticipantFrom(
			getContext(c), participantIDs, filter.BeginID,
		)
		if err != nil {
			return err
		}
		solutions = participantSolutions
	}
	defer func() { _ = solutions.Close() }()
	var resp ContestSolutions
	solutionsCount := 0
	for solutions.Next() {
		solution := solutions.Row()
		if solutionsCount >= maxSolutionLimit ||
			len(resp.Solutions) >= filter.Limit {
			resp.NextBeginID = solution.ID
			break
		}
		solutionsCount++
		if !filter.Filter(solution) {
			continue
		}
		permissions := v.getContestSolutionPermissions(contestCtx, solution)
		if permissions.HasPermission(perms.ObserveContestSolutionRole) {
			resp.Solutions = append(
				resp.Solutions,
				v.makeContestSolution(c, solution, false),
			)
		}
	}
	if err := solutions.Err(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeContestSolution(c echo.Context) error {
	solution, ok := c.Get(contestSolutionKey).(models.ContestSolution)
	if !ok {
		return fmt.Errorf("solution not extracted")
	}
	resp := v.makeContestSolution(c, solution, true)
	return c.JSON(http.StatusOK, resp)
}

func (v *View) rejudgeContestSolution(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contestSolution, ok := c.Get(contestSolutionKey).(models.ContestSolution)
	if !ok {
		return fmt.Errorf("solution not extracted")
	}
	solution, err := v.core.Solutions.Get(getContext(c), contestSolution.ID)
	if err != nil {
		return err
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		if err := solution.SetReport(nil); err != nil {
			return err
		}
		if err := v.core.Solutions.Update(ctx, solution); err != nil {
			return err
		}
		task := models.Task{}
		if err := task.SetConfig(models.JudgeSolutionTaskConfig{
			SolutionID:   solution.ID,
			EnablePoints: getEnablePoints(contestCtx),
		}); err != nil {
			return err
		}
		return v.core.Tasks.Create(ctx, &task)
	}, sqlRepeatableRead); err != nil {
		return err
	}
	resp := v.makeContestSolution(c, contestSolution, true)
	resp.Solution.Report = &SolutionReport{
		Verdict: models.QueuedTask.String(),
	}
	return c.JSON(http.StatusOK, resp)
}

type ContestSolution struct {
	ID          int64               `json:"id"`
	ContestID   int64               `json:"contest_id"`
	Solution    Solution            `json:"solution"`
	Problem     *ContestProblem     `json:"problem,omitempty"`
	Participant *ContestParticipant `json:"participant,omitempty"`
}

type SubmitSolutionForm struct {
	CompilerID int64   `form:"compiler_id" json:"compiler_id"`
	Content    *string `form:"content" json:"content,omitempty"`
	// ContentFile will be initialized with the content if it is provided.
	ContentFile *FileReader `json:"-"`
}

func (f *SubmitSolutionForm) Parse(c echo.Context) error {
	if err := c.Bind(f); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if f.Content != nil {
		content := bytes.NewReader([]byte(*f.Content))
		file := FileReader{
			Reader: content,
			Size:   int64(content.Len()),
		}
		f.ContentFile = &file
	} else {
		formFile, err := c.FormFile("file")
		if err != nil {
			return err
		}
		file, err := managers.NewMultipartFileReader(formFile)
		if err != nil {
			return err
		}
		f.ContentFile = file
	}
	return nil
}

func (v *View) hasSolutionsQuota(
	contestCtx *managers.ContestContext,
	participant models.ContestParticipant,
	logger echo.Logger,
) bool {
	if participant.Kind == models.ManagerParticipant {
		return true
	}
	solutions, err := v.core.ContestSolutions.FindByParticipant(contestCtx, participant.ID)
	if err != nil {
		logger.Warn("Cannot get solutions for participant: %v", participant.ID)
		return false
	}
	defer func() { _ = solutions.Close() }()
	window := v.getInt64Setting("contests.solutions_quota.window", logger).OrElse(60)
	amount := v.getInt64Setting("contests.solutions_quota.amount", logger).OrElse(3)
	toTime := contestCtx.Now
	fromTime := toTime.Add(-time.Second * time.Duration(window))
	for solutions.Next() {
		contestSolution := solutions.Row()
		solution, err := v.core.Solutions.Get(contestCtx, contestSolution.ID)
		if err != nil {
			logger.Warn("Cannot find solution: %v", contestSolution.ID)
			continue
		}
		createTime := time.Unix(solution.CreateTime, 0)
		if createTime.Before(fromTime) {
			continue
		}
		if createTime.After(toTime) {
			continue
		}
		amount--
		if amount <= 0 {
			return false
		}
	}
	return true
}

func (v *View) submitContestProblemSolution(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	problem, ok := c.Get(contestProblemKey).(models.ContestProblem)
	if !ok {
		return fmt.Errorf("contest problem not extracted")
	}
	account := contestCtx.Account
	if account == nil {
		return fmt.Errorf("account not extracted")
	}
	participant := contestCtx.GetEffectiveParticipant()
	if participant == nil {
		return errorResponse{
			Code:    http.StatusForbidden,
			Message: localize(c, "Participant not found."),
		}
	}
	if !contestCtx.HasEffectivePermission(perms.SubmitContestSolutionRole) {
		return errorResponse{
			Code:               http.StatusForbidden,
			Message:            localize(c, "Account missing permissions."),
			MissingPermissions: []string{perms.SubmitContestSolutionRole},
		}
	}
	needUpdate := false
	if participant.Kind == models.RegularParticipant &&
		contestCtx.Stage == managers.ContestStarted {
		contestConfig, err := contestCtx.Contest.GetConfig()
		if err != nil {
			return err
		}
		var config models.RegularParticipantConfig
		if err := participant.ScanConfig(&config); err != nil {
			return err
		}
		if config.BeginTime != contestConfig.BeginTime {
			config.BeginTime = contestConfig.BeginTime
			if err := participant.SetConfig(config); err != nil {
				return err
			}
			needUpdate = true
		}
	}
	if participant.ID == 0 {
		if err := v.core.ContestParticipants.Create(
			getContext(c), participant,
		); err != nil {
			return err
		}
	} else if needUpdate {
		if err := v.core.ContestParticipants.Update(
			getContext(c), *participant,
		); err != nil {
			return err
		}
	}
	if participant.ID == 0 {
		return fmt.Errorf("unable to register participant")
	}
	if !v.hasSolutionsQuota(contestCtx, *participant, c.Logger()) {
		return errorResponse{
			Code:    http.StatusTooManyRequests,
			Message: localize(c, "Too many requests."),
		}
	}
	var form SubmitSolutionForm
	if err := form.Parse(c); err != nil {
		return err
	}
	defer func() { _ = form.ContentFile.Close() }()
	if form.ContentFile.Size <= 0 {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "File is empty."),
		}
	}
	if form.ContentFile.Size >= 256*1024 {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "File is too large."),
		}
	}
	if _, err := v.core.Compilers.Get(getContext(c), form.CompilerID); err != nil {
		if err == sql.ErrNoRows {
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Compiler not found."),
			}
		}
		return err
	}
	solution := models.Solution{
		Kind:       models.ContestSolutionKind,
		ProblemID:  problem.ProblemID,
		AuthorID:   account.ID,
		CompilerID: form.CompilerID,
		CreateTime: contestCtx.Now.Unix(),
	}
	contestSolution := models.ContestSolution{
		ContestID:     contest.ID,
		ParticipantID: participant.ID,
		ProblemID:     problem.ID,
	}
	file, err := v.files.UploadFile(getContext(c), form.ContentFile)
	if err != nil {
		return err
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		if err := v.files.ConfirmUploadFile(ctx, &file); err != nil {
			return err
		}
		solution.ContentID = models.NInt64(file.ID)
		if err := v.core.Solutions.Create(ctx, &solution); err != nil {
			return err
		}
		contestSolution.ID = solution.ID
		if err := v.core.ContestSolutions.Create(
			ctx, &contestSolution,
		); err != nil {
			return err
		}
		task := models.Task{}
		if err := task.SetConfig(models.JudgeSolutionTaskConfig{
			SolutionID:   solution.ID,
			EnablePoints: getEnablePoints(contestCtx),
		}); err != nil {
			return err
		}
		return v.core.Tasks.Create(ctx, &task)
	}, sqlRepeatableRead); err != nil {
		return err
	}
	return c.JSON(
		http.StatusCreated,
		v.makeContestSolution(c, contestSolution, true),
	)
}

func getEnablePoints(ctx *managers.ContestContext) bool {
	return ctx.ContestConfig.StandingsKind == models.IOIStandings
}

func (v *View) makeContestSolution(
	c echo.Context, solution models.ContestSolution, withLogs bool,
) ContestSolution {
	resp := ContestSolution{
		ID:        solution.ID,
		ContestID: solution.ContestID,
	}
	if baseSolution, err := v.core.Solutions.Get(
		getContext(c), solution.ID,
	); err == nil {
		resp.Solution = v.makeSolution(c, baseSolution, withLogs)
		resp.Solution.Problem = nil
		resp.Solution.User = nil
		resp.Solution.ScopeUser = nil
	}
	if problem, err := v.core.ContestProblems.Get(
		getContext(c), solution.ProblemID,
	); err == nil {
		problemResp := v.makeContestProblem(c, problem, false)
		resp.Problem = &problemResp
	}
	if participant, err := v.core.ContestParticipants.Get(
		getContext(c), solution.ParticipantID,
	); err == nil {
		participantResp := makeContestParticipant(c, participant, v.core)
		resp.Participant = &participantResp
	}
	return resp
}

func makeContestParticipant(
	c echo.Context,
	participant models.ContestParticipant,
	core *core.Core,
) ContestParticipant {
	ctx := getContext(c)
	resp := ContestParticipant{
		ID:        participant.ID,
		ContestID: participant.ContestID,
		Kind:      participant.Kind,
	}
	if account, err := core.Accounts.Get(
		ctx, participant.AccountID,
	); err == nil {
		switch account.Kind {
		case models.UserAccountKind:
			if user, err := core.Users.Get(ctx, account.ID); err == nil {
				resp.User = &User{
					ID:    user.ID,
					Login: user.Login,
				}
			}
		case models.ScopeUserAccountKind:
			if user, err := core.ScopeUsers.Get(ctx, account.ID); err == nil {
				resp.ScopeUser = &ScopeUser{
					ID:    user.ID,
					Login: user.Login,
					Title: string(user.Title),
				}
			}
		case models.ScopeAccountKind:
			if scope, err := core.Scopes.Get(ctx, account.ID); err == nil {
				resp.Scope = &Scope{
					ID:    scope.ID,
					Title: scope.Title,
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
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Invalid contest ID."),
			}
		}
		if err := syncStore(c, v.core.Contests); err != nil {
			return err
		}
		contest, err := v.core.Contests.Get(getContext(c), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: localize(c, "Contest not found."),
				}
			}
			return err
		}
		accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
		if !ok {
			return fmt.Errorf("account not extracted")
		}
		contestCtx, err := v.contests.BuildContext(accountCtx, contest)
		if err != nil {
			return err
		}
		c.Set(contestCtxKey, contestCtx)
		c.Set(permissionCtxKey, contestCtx)
		return next(c)
	}
}

func (v *View) extractContestProblem(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := c.Param("problem")
		if len(code) == 0 {
			return errorResponse{
				Code:    http.StatusNotFound,
				Message: localize(c, "Empty problem code."),
			}
		}
		contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
		if !ok {
			return fmt.Errorf("contest not extracted")
		}
		contest := contestCtx.Contest
		if err := syncStore(c, v.core.ContestProblems); err != nil {
			return err
		}
		if id, err := strconv.ParseInt(code, 10, 64); err == nil {
			contestProblem, err := v.core.ContestProblems.Get(getContext(c), id)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err == nil && contestProblem.ContestID == contest.ID {
				problem, err := v.core.Problems.Get(getContext(c), contestProblem.ProblemID)
				if err != nil {
					return err
				}
				c.Set(contestProblemKey, contestProblem)
				c.Set(problemKey, problem)
				return next(c)
			}
		}
		contestProblem, err := findContestProblem(getContext(c), v.core, contest.ID, code)
		if err != nil {
			return err
		}
		if contestProblem == nil {
			return errorResponse{
				Code: http.StatusNotFound,
				Message: localize(
					c, "Problem {code} does not exists.",
					replaceField("code", code),
				),
			}
		}
		problem, err := v.core.Problems.Get(getContext(c), contestProblem.ProblemID)
		if err != nil {
			return err
		}
		c.Set(contestProblemKey, *contestProblem)
		c.Set(problemKey, problem)
		return next(c)
	}
}

func findContestProblem(ctx context.Context, c *core.Core, contestID int64, code string) (*models.ContestProblem, error) {
	rows, err := c.ContestProblems.FindByContest(ctx, contestID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		row := rows.Row()
		if row.Code == code {
			return &row, nil
		}
	}
	return nil, rows.Err()
}

func (v *View) extractContestParticipant(
	next echo.HandlerFunc,
) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("participant"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Invalid participant ID."),
			}
		}
		if err := syncStore(c, v.core.ContestParticipants); err != nil {
			return err
		}
		participant, err := v.core.ContestParticipants.Get(getContext(c), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: localize(c, "Participant not found."),
				}
			}
			return err
		}
		contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
		if !ok {
			return fmt.Errorf("contest not extracted")
		}
		if contestCtx.Contest.ID != participant.ContestID {
			return errorResponse{
				Code:    http.StatusNotFound,
				Message: localize(c, "Participant not found."),
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
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Invalid solution ID."),
			}
		}
		if err := syncStore(c, v.core.ContestSolutions); err != nil {
			return err
		}
		if err := syncStore(c, v.core.Solutions); err != nil {
			return err
		}
		solution, err := v.core.ContestSolutions.Get(getContext(c), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: localize(c, "Solution not found."),
				}
			}
			return err
		}
		contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
		if !ok {
			return fmt.Errorf("contest not extracted")
		}
		if contestCtx.Contest.ID != solution.ContestID {
			return errorResponse{
				Code:    http.StatusNotFound,
				Message: localize(c, "Solution not found."),
			}
		}
		c.Set(contestSolutionKey, solution)
		c.Set(
			permissionCtxKey,
			v.getContestSolutionPermissions(contestCtx, solution),
		)
		return next(c)
	}
}

func (v *View) getContestSolutionPermissions(
	ctx *managers.ContestContext, solution models.ContestSolution,
) perms.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if solution.ID == 0 {
		return permissions
	}
	if solution.ParticipantID != 0 {
		for _, participant := range ctx.Participants {
			if participant.ID == solution.ParticipantID {
				permissions.AddPermission(perms.ObserveContestSolutionRole)
				break
			}
		}
	}
	return permissions
}

func contestProblemLess(l, r ContestProblem) bool {
	if l.ContestID == r.ContestID {
		return l.Code < r.Code
	}
	return l.ID < r.ID
}

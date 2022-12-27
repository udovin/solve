package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

func (v *View) registerContestHandlers(g *echo.Group) {
	g.GET(
		"/v0/contests", v.observeContests,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(models.ObserveContestsRole),
	)
	g.POST(
		"/v0/contests", v.createContest,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(models.CreateContestRole),
	)
	g.GET(
		"/v0/contests/:contest", v.observeContest,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractContest,
		v.requirePermission(models.ObserveContestRole),
	)
	g.PATCH(
		"/v0/contests/:contest", v.updateContest,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(models.UpdateContestRole),
	)
	g.DELETE(
		"/v0/contests/:contest", v.deleteContest,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(models.DeleteContestRole),
	)
	g.GET(
		"/v0/contests/:contest/problems", v.observeContestProblems,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractContest,
		v.requirePermission(models.ObserveContestProblemsRole),
	)
	g.GET(
		"/v0/contests/:contest/problems/:problem", v.observeContestProblem,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.extractContest, v.extractContestProblem,
		v.requirePermission(models.ObserveContestProblemRole),
	)
	g.GET(
		"/v0/contests/:contest/problems/:problem/resources/:resource",
		v.observeProblemResource,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.extractContest, v.extractContestProblem,
		v.requirePermission(models.ObserveContestProblemRole),
	)
	g.POST(
		"/v0/contests/:contest/problems", v.createContestProblem,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(models.CreateContestProblemRole),
	)
	g.DELETE(
		"/v0/contests/:contest/problems/:problem", v.deleteContestProblem,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.extractContestProblem,
		v.requirePermission(models.DeleteContestProblemRole),
	)
	g.POST(
		"/v0/contests/:contest/problems/:problem/submit",
		v.submitContestProblemSolution, v.extractAuth(v.sessionAuth),
		v.extractContest, v.extractContestProblem,
		v.requirePermission(models.SubmitContestSolutionRole),
	)
	g.GET(
		"/v0/contests/:contest/solutions", v.observeContestSolutions,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractContest,
		v.requirePermission(models.ObserveContestSolutionsRole),
	)
	g.GET(
		"/v0/contests/:contest/solutions/:solution", v.observeContestSolution,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.extractContest, v.extractContestSolution,
		v.requirePermission(models.ObserveContestSolutionRole),
	)
	g.GET(
		"/v0/contests/:contest/participants", v.observeContestParticipants,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractContest,
		v.requirePermission(models.ObserveContestParticipantsRole),
	)
	g.POST(
		"/v0/contests/:contest/participants", v.createContestParticipant,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(models.CreateContestParticipantRole),
	)
	g.DELETE(
		"/v0/contests/:contest/participants/:participant",
		v.deleteContestParticipant, v.extractAuth(v.sessionAuth),
		v.extractContest, v.extractContestParticipant,
		v.requirePermission(models.DeleteContestParticipantRole),
	)
	g.POST(
		"/v0/contests/:contest/register", v.registerContest,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(models.RegisterContestRole),
	)
}

type Contest struct {
	ID                 int64    `json:"id"`
	Title              string   `json:"title"`
	BeginTime          NInt64   `json:"begin_time,omitempty"`
	Duration           int      `json:"duration,omitempty"`
	Permissions        []string `json:"permissions,omitempty"`
	EnableRegistration bool     `json:"enable_registration"`
	EnableUpsolving    bool     `json:"enable_upsolving"`
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

var contestPermissions = []string{
	models.UpdateContestRole,
	models.DeleteContestRole,
	models.RegisterContestRole,
	models.DeregisterContestRole,
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

func makeContest(
	contest models.Contest,
	permissions managers.Permissions,
	core *core.Core,
) Contest {
	resp := Contest{ID: contest.ID, Title: contest.Title}
	if config, err := contest.GetConfig(); err == nil {
		resp.BeginTime = config.BeginTime
		resp.Duration = config.Duration
		resp.EnableRegistration = config.EnableRegistration
		resp.EnableUpsolving = config.EnableUpsolving
	}
	for _, permission := range contestPermissions {
		if permissions.HasPermission(permission) {
			resp.Permissions = append(resp.Permissions, permission)
		}
	}
	return resp
}

func (v *View) makeContestProblem(
	c echo.Context, contestProblem models.ContestProblem, withStatement bool,
) ContestProblem {
	resp := ContestProblem{
		ContestID: contestProblem.ContestID,
		Code:      contestProblem.Code,
	}
	if problem, err := v.core.Problems.Get(
		contestProblem.ProblemID,
	); err == nil {
		resp.Problem = v.makeProblem(
			c, problem, managers.PermissionSet{}, withStatement,
		)
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
	contests, err := v.core.Contests.All()
	if err != nil {
		return err
	}
	for _, contest := range contests {
		if !filter.Filter(contest) {
			continue
		}
		contestCtx, err := v.contests.BuildContext(accountCtx, contest)
		if err != nil {
			return err
		}
		if contestCtx.HasPermission(models.ObserveContestRole) {
			resp.Contests = append(
				resp.Contests,
				makeContest(contest, contestCtx, v.core),
			)
		}
	}
	sortFunc(resp.Contests, contestGreater)
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeContest(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	return c.JSON(http.StatusOK, makeContest(contest, contestCtx, v.core))
}

type updateContestForm struct {
	Title              *string `json:"title" form:"title"`
	BeginTime          *NInt64 `json:"begin_time" form:"begin_time"`
	Duration           *int    `json:"duration" form:"duration"`
	EnableRegistration *bool   `json:"enable_registration" form:"enable_registration"`
	EnableUpsolving    *bool   `json:"enable_upsolving" form:"enable_upsolving"`
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
	return c.JSON(http.StatusCreated, makeContest(contest, accountCtx, nil))
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
	if err := v.core.Contests.Update(getContext(c), contest); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, makeContest(contest, contestCtx, v.core))
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
	return c.JSON(http.StatusOK, makeContest(contest, contestCtx, nil))
}

func (v *View) observeContestProblems(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	problems, err := v.core.ContestProblems.FindByContest(contest.ID)
	if err != nil {
		return err
	}
	resp := ContestProblems{}
	for _, problem := range problems {
		resp.Problems = append(
			resp.Problems,
			v.makeContestProblem(c, problem, false),
		)
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

type createContestProblemForm struct {
	Code      string `json:"code"`
	ProblemID int64  `json:"problem_id"`
}

func (f createContestProblemForm) Update(
	c echo.Context,
	problem *models.ContestProblem,
	problems *models.ProblemStore,
) *errorResponse {
	errors := errorFields{}
	if len(f.Code) == 0 {
		errors["code"] = errorField{
			Message: localize(c, "Code is empty."),
		}
	}
	if len(f.Code) > 4 {
		errors["code"] = errorField{
			Message: localize(c, "Code is too long."),
		}
	}
	if len(errors) > 0 {
		return &errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	if _, err := problems.Get(f.ProblemID); err != nil {
		return &errorResponse{
			Code: http.StatusNotFound,
			Message: localize(
				c, "Problem {id} does not exists.",
				replaceField("id", f.ProblemID),
			),
		}
	}
	problem.Code = f.Code
	problem.ProblemID = f.ProblemID
	return nil
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
	{
		problems, err := v.core.ContestProblems.FindByContest(contest.ID)
		if err != nil {
			return err
		}
		for _, contestProblem := range problems {
			if problem.Code == contestProblem.Code {
				return errorResponse{
					Code: http.StatusBadRequest,
					Message: localize(
						c, "Problem with code {code} already exists.",
						replaceField("code", problem.Code),
					),
				}
			}
			if problem.ProblemID == contestProblem.ProblemID {
				return errorResponse{
					Code: http.StatusBadRequest,
					Message: localize(
						c, "Problem {id} already exists.",
						replaceField("id", problem.ProblemID),
					),
				}
			}
		}
	}
	if err := v.core.ContestProblems.Create(
		getContext(c), &problem,
	); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, v.makeContestProblem(c, problem, false))
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
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
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
	c echo.Context, participant *models.ContestParticipant, core *core.Core,
) *errorResponse {
	if f.UserID != nil {
		user, err := core.Users.Get(*f.UserID)
		if err != nil {
			return &errorResponse{
				Code: http.StatusBadRequest,
				Message: localize(
					c, "User {id} does not exists.",
					replaceField("id", *f.UserID),
				),
			}
		}
		participant.AccountID = user.AccountID
	} else if f.UserLogin != nil {
		user, err := core.Users.GetByLogin(*f.UserLogin)
		if err != nil {
			return &errorResponse{
				Code: http.StatusBadRequest,
				Message: localize(
					c, "User \"{login}\" does not exists.",
					replaceField("login", *f.UserLogin),
				),
			}
		}
		participant.AccountID = user.AccountID
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
	var form createContestParticipantForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var participant models.ContestParticipant
	if err := form.Update(c, &participant, v.core); err != nil {
		return err
	}
	participant.ContestID = contest.ID
	{
		participants, err := v.core.ContestParticipants.FindByContestAccount(
			contest.ID, participant.AccountID,
		)
		if err != nil {
			return err
		}
		for _, p := range participants {
			if p.Kind == participant.Kind {
				return errorResponse{
					Code: http.StatusBadRequest,
					Message: localize(
						c, "Participant with {kind} kind already exists.",
						replaceField("kind", p.Kind),
					),
				}
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
	Solutions []ContestSolution `json:"solutions"`
}

func (v *View) observeContestSolutions(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	if err := syncStore(c, v.core.Solutions); err != nil {
		return err
	}
	if err := syncStore(c, v.core.ContestSolutions); err != nil {
		return err
	}
	solutions, err := v.core.ContestSolutions.FindByContest(contest.ID)
	if err != nil {
		return err
	}
	var resp ContestSolutions
	for _, solution := range solutions {
		permissions := v.getContestSolutionPermissions(contestCtx, solution)
		if permissions.HasPermission(models.ObserveContestSolutionRole) {
			resp.Solutions = append(
				resp.Solutions,
				v.makeContestSolution(c, solution, false),
			)
		}
	}
	sortFunc(resp.Solutions, contestSolutionGreater)
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

type TestReport struct {
	Verdict  models.Verdict `json:"verdict"`
	CheckLog string         `json:"check_log,omitempty"`
	Input    string         `json:"input,omitempty"`
	Output   string         `json:"output,omitempty"`
}

type SolutionReport struct {
	Verdict    string       `json:"verdict"`
	Tests      []TestReport `json:"tests,omitempty"`
	CompileLog string       `json:"compile_log,omitempty"`
}

type ContestSolution struct {
	ID          int64               `json:"id"`
	ContestID   int64               `json:"contest_id"`
	Problem     *ContestProblem     `json:"problem,omitempty"`
	Compiler    *Compiler           `json:"compiler,omitempty"`
	Participant *ContestParticipant `json:"participant,omitempty"`
	Content     string              `json:"content,omitempty"`
	Report      *SolutionReport     `json:"report"`
	CreateTime  int64               `json:"create_time"`
}

type SubmitSolutionForm struct {
	CompilerID  int64       `form:"compiler_id" json:"compiler_id"`
	ContentFile *FileReader `json:"-"`
}

func (f *SubmitSolutionForm) Parse(c echo.Context) error {
	if err := c.Bind(f); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	formFile, err := c.FormFile("file")
	if err != nil {
		return err
	}
	file, err := managers.NewMultipartFileReader(formFile)
	if err != nil {
		return err
	}
	f.ContentFile = file
	return nil
}

func (v *View) submitContestProblemSolution(c echo.Context) error {
	now := getNow(c)
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
	if !contestCtx.HasEffectivePermission(models.SubmitContestSolutionRole) {
		return errorResponse{
			Code:               http.StatusForbidden,
			Message:            localize(c, "Account missing permissions."),
			MissingPermissions: []string{models.SubmitContestSolutionRole},
		}
	}
	if participant.ID == 0 {
		if err := v.core.ContestParticipants.Create(
			getContext(c), participant,
		); err != nil {
			return err
		}
	}
	if participant.ID == 0 {
		return fmt.Errorf("unable to register participant")
	}
	var form SubmitSolutionForm
	if err := form.Parse(c); err != nil {
		return err
	}
	defer func() { _ = form.ContentFile.Close() }()
	if _, err := v.core.Compilers.Get(form.CompilerID); err != nil {
		if err == sql.ErrNoRows {
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Compiler not found."),
			}
		}
		return err
	}
	solution := models.Solution{
		ProblemID:  problem.ProblemID,
		AuthorID:   account.ID,
		CompilerID: form.CompilerID,
		CreateTime: now.Unix(),
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
		contestSolution.SolutionID = solution.ID
		if err := v.core.ContestSolutions.Create(
			ctx, &contestSolution,
		); err != nil {
			return err
		}
		task := models.Task{}
		if err := task.SetConfig(models.JudgeSolutionTaskConfig{
			SolutionID: solution.ID,
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

func (v *View) makeContestSolution(
	c echo.Context, solution models.ContestSolution, withLogs bool,
) ContestSolution {
	resp := ContestSolution{
		ID:        solution.ID,
		ContestID: solution.ContestID,
	}
	if baseSolution, err := v.core.Solutions.Get(
		solution.SolutionID,
	); err == nil {
		resp.CreateTime = baseSolution.CreateTime
		if withLogs {
			resp.Content = v.makeSolutionContent(c, baseSolution)
		}
		resp.Report = v.makeSolutionReport(c, baseSolution, withLogs)
		if compiler, err := v.core.Compilers.Get(
			baseSolution.CompilerID,
		); err == nil {
			compilerResp := makeCompiler(compiler)
			resp.Compiler = &compilerResp
		}
	}
	if problem, err := v.core.ContestProblems.Get(
		solution.ProblemID,
	); err == nil {
		problemResp := v.makeContestProblem(c, problem, false)
		resp.Problem = &problemResp
	}
	if participant, err := v.core.ContestParticipants.Get(
		solution.ParticipantID,
	); err == nil {
		participantResp := makeContestParticipant(c, participant, v.core)
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
	if account, err := core.Accounts.Get(participant.AccountID); err == nil {
		switch account.Kind {
		case models.UserAccount:
			if user, err := core.Users.GetByAccount(account.ID); err == nil {
				resp.User = &User{ID: user.ID, Login: user.Login}
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
		contest, err := v.core.Contests.Get(id)
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
		problems, err := v.core.ContestProblems.FindByContest(contest.ID)
		if err != nil {
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
			return errorResponse{
				Code: http.StatusNotFound,
				Message: localize(
					c, "Problem {code} does not exists.",
					replaceField("code", code),
				),
			}
		}
		problem, err := v.core.Problems.Get(problems[pos].ProblemID)
		if err != nil {
			return err
		}
		c.Set(contestProblemKey, problems[pos])
		c.Set(problemKey, problem)
		return next(c)
	}
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
		participant, err := v.core.ContestParticipants.Get(id)
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
		if err := syncStore(c, v.core.Solutions); err != nil {
			return err
		}
		if err := syncStore(c, v.core.ContestSolutions); err != nil {
			return err
		}
		solution, err := v.core.ContestSolutions.Get(id)
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
) managers.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if solution.ID == 0 {
		return permissions
	}
	if solution.ParticipantID != 0 {
		for _, participant := range ctx.Participants {
			if participant.ID == solution.ParticipantID {
				permissions[models.ObserveContestSolutionRole] = struct{}{}
				break
			}
		}
	}
	return permissions
}

func contestGreater(l, r Contest) bool {
	return l.ID > r.ID
}

func contestSolutionGreater(l, r ContestSolution) bool {
	return l.ID > r.ID
}

func contestProblemLess(l, r ContestProblem) bool {
	if l.ContestID == r.ContestID {
		return l.Code < r.Code
	}
	return l.ID < r.ID
}

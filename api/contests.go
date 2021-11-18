package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

func (v *View) registerContestHandlers(g *echo.Group) {
	g.GET(
		"/contests", v.observeContests,
		v.sessionAuth,
		v.requireAuthRole(models.ObserveContestsRole),
	)
	g.POST(
		"/contests", v.createContest,
		v.sessionAuth,
		v.requireAuthRole(models.CreateContestRole),
	)
	g.GET(
		"/contests/:contest", v.observeContest,
		v.sessionAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.ObserveContestRole),
	)
	g.DELETE(
		"/contests/:contest", v.deleteContest,
		v.sessionAuth, v.requireAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.DeleteContestRole),
	)
	g.GET(
		"/contests/:contest/problems", v.observeContestProblems,
		v.sessionAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.ObserveContestProblemsRole),
	)
	g.GET(
		"/contests/:contest/problems/:problem", v.observeContestProblem,
		v.sessionAuth, v.extractContest, v.extractContestProblem, v.extractContestRoles,
		v.requireAuthRole(models.ObserveContestProblemRole),
	)
	g.POST(
		"/contests/:contest/problems", v.createContestProblem,
		v.sessionAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.CreateContestProblemRole),
	)
	g.DELETE(
		"/contests/:contest/problems/:problem", v.deleteContestProblem,
		v.sessionAuth, v.extractContest, v.extractContestProblem, v.extractContestRoles,
		v.requireAuthRole(models.DeleteContestProblemRole),
	)
	g.GET(
		"/contests/:contest/participants", v.observeContestParticipants,
		v.sessionAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.ObserveContestParticipantsRole),
	)
	g.POST(
		"/contests/:contest/participants", v.createContestParticipant,
		v.sessionAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.CreateContestParticipantRole),
	)
	g.DELETE(
		"/contests/:contest/participants/:participant", v.deleteContestParticipant,
		v.sessionAuth, v.extractContest, v.extractContestParticipant, v.extractContestRoles,
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

var contestPermissions = []string{
	models.UpdateContestRole,
	models.DeleteContestRole,
	models.ObserveContestProblemsRole,
	models.CreateContestProblemRole,
	models.DeleteContestProblemRole,
	models.ObserveContestParticipantsRole,
	models.CreateContestParticipantRole,
	models.DeleteContestParticipantRole,
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

type createContestForm struct {
	Title string `json:"title"`
}

func (f createContestForm) validate() *errorResponse {
	errors := errorFields{}
	if len(f.Title) < 4 {
		errors["title"] = errorField{Message: "title is too short"}
	}
	if len(f.Title) > 64 {
		errors["title"] = errorField{Message: "title is too long"}
	}
	if len(errors) > 0 {
		return &errorResponse{
			Message:       "form has invalid fields",
			InvalidFields: errors,
		}
	}
	return nil
}

func (f createContestForm) Update(contest *models.Contest) *errorResponse {
	if err := f.validate(); err != nil {
		return err
	}
	contest.Title = f.Title
	return nil
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
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		var err error
		contest, err = v.core.Contests.CreateTx(tx, contest)
		return err
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusCreated, makeContest(contest, nil, nil))
}

func (v *View) deleteContest(c echo.Context) error {
	contest, ok := c.Get(contestKey).(models.Contest)
	if !ok {
		c.Logger().Error("contest not extracted")
		return fmt.Errorf("contest not extracted")
	}
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		return v.core.Contests.DeleteTx(tx, contest.ID)
	}); err != nil {
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
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		var err error
		problem, err = v.core.ContestProblems.CreateTx(
			tx, problem,
		)
		return err
	}); err != nil {
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
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		return v.core.ContestProblems.DeleteTx(tx, problem.ID)
	}); err != nil {
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
}

type ContestParticipants struct {
	Participants []ContestParticipant `json:"participants"`
}

func (v *View) observeContestParticipants(c echo.Context) error {
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
	participants, err := v.core.ContestParticipants.FindByContest(contest.ID)
	if err != nil {
		return err
	}
	var resp ContestParticipants
	for _, participant := range participants {
		resp.Participants = append(
			resp.Participants,
			makeContestParticipant(c, participant, roles, v.core),
		)
	}
	return c.JSON(http.StatusOK, resp)
}

type createContestParticipantForm struct {
	UserID *int64 `json:"user_id"`
}

func (f createContestParticipantForm) Update(
	participant *models.ContestParticipant, core *core.Core,
) *errorResponse {
	if f.UserID != nil {
		user, err := core.Users.Get(*f.UserID)
		if err != nil {
			return &errorResponse{Message: fmt.Sprintf(
				"user %d does not exists", *f.UserID,
			)}
		}
		participant.AccountID = user.AccountID
	}
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
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		return v.core.ContestParticipants.CreateTx(tx, &participant)
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(
		http.StatusCreated,
		makeContestParticipant(c, participant, nil, nil),
	)
}

func (v *View) deleteContestParticipant(c echo.Context) error {
	participant, ok := c.Get(contestParticipantKey).(models.ContestParticipant)
	if !ok {
		c.Logger().Error("contest participant not extracted")
		return fmt.Errorf("contest participant not extracted")
	}
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		return v.core.ContestParticipants.DeleteTx(tx, participant.ID)
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(
		http.StatusOK,
		makeContestParticipant(c, participant, nil, nil),
	)
}

func makeContestParticipant(
	c echo.Context, participant models.ContestParticipant,
	roles core.RoleSet, core *core.Core,
) ContestParticipant {
	resp := ContestParticipant{
		ID:        participant.ID,
		ContestID: participant.ContestID,
	}
	if core != nil {
		if account, err := core.Accounts.Get(participant.AccountID); err == nil {
			switch account.Kind {
			case models.UserAccount:
				if user, err := core.Users.GetByAccount(account.ID); err == nil {
					userResp := makeUser(c, user, roles, core)
					resp.User = &userResp
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
			return err
		}
		contest, err := v.core.Contests.Get(id)
		if err == sql.ErrNoRows {
			if err := v.core.Contests.SyncTx(v.core.DB); err != nil {
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
			return err
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

func (v *View) extendContestRoles(
	c echo.Context, roles core.RoleSet, contest models.Contest,
) core.RoleSet {
	contestRoles := roles.Clone()
	addRole := func(code string) {
		if err := v.core.AddRole(contestRoles, code); err != nil {
			c.Logger().Error(err)
		}
	}
	account, ok := c.Get(authAccountKey).(models.Account)
	if ok && contest.OwnerID != 0 && account.ID == int64(contest.OwnerID) {
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
		participants, err := v.core.ContestParticipants.
			FindByContestAccount(contest.ID, account.ID)
		if err != nil {
			c.Logger().Error(err)
		} else if len(participants) > 0 {
			addRole(models.ObserveContestRole)
			// TODO(iudovin): Add support of start time.
			addRole(models.ObserveContestProblemsRole)
			addRole(models.ObserveContestProblemRole)
		}
	}
	return contestRoles
}

func (v *View) extractContestRoles(next echo.HandlerFunc) echo.HandlerFunc {
	nextWrap := func(c echo.Context) error {
		contest, ok := c.Get(contestKey).(models.Contest)
		if !ok {
			c.Logger().Error("session not extracted")
			return fmt.Errorf("session not extracted")
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

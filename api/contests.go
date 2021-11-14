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
	models.ObserveContestProblemRole,
	models.CreateContestProblemRole,
	models.DeleteContestProblemRole,
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
	for _, contestProblem := range problems {
		problem, err := v.core.Problems.Get(contestProblem.ProblemID)
		if err != nil {
			return err
		}
		resp.Problems = append(resp.Problems, ContestProblem{
			ContestID: contestProblem.ContestID,
			Code:      contestProblem.Code,
			Problem:   makeProblem(problem),
		})
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeContestProblem(c echo.Context) error {
	contestProblem, ok := c.Get(contestProblemKey).(models.ContestProblem)
	if !ok {
		c.Logger().Error("contest problem not extracted")
		return fmt.Errorf("contest problem not extracted")
	}
	problem, err := v.core.Problems.Get(contestProblem.ProblemID)
	if err != nil {
		return err
	}
	resp := ContestProblem{
		ContestID: contestProblem.ContestID,
		Code:      contestProblem.Code,
		Problem:   makeProblem(problem),
	}
	return c.JSON(http.StatusOK, resp)
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
	var contestProblem models.ContestProblem
	if err := form.Update(&contestProblem, v.core.Problems); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	contestProblem.ContestID = contest.ID
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		var err error
		contestProblem, err = v.core.ContestProblems.CreateTx(
			tx, contestProblem,
		)
		return err
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusCreated, makeContest(contest, nil, nil))
}

func (v *View) deleteContestProblem(c echo.Context) error {
	contestProblem, ok := c.Get(contestProblemKey).(models.ContestProblem)
	if !ok {
		c.Logger().Error("contest problem not extracted")
		return fmt.Errorf("contest problem not extracted")
	}
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		return v.core.ContestProblems.DeleteTx(tx, contestProblem.ID)
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	resp := ContestProblem{
		ContestID: contestProblem.ContestID,
		Code:      contestProblem.Code,
		Problem:   Problem{ID: contestProblem.ProblemID},
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) extractContest(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("contest"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return err
		}
		contest, err := v.core.Contests.Get(id)
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
		code := c.Param("code")
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

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
	g.GET(
		"/contests/:contest/problems/:problem", v.observeContestProblem,
		v.sessionAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.ObserveContestProblemsRole),
	)
	g.DELETE(
		"/contests/:contest", v.deleteContest,
		v.sessionAuth, v.requireAuth, v.extractContest, v.extractContestRoles,
		v.requireAuthRole(models.DeleteContestRole),
	)
}

type Contest struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
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

type Contests struct {
	Contests []Contest `json:"contests"`
}

func makeContest(contest models.Contest) Contest {
	resp := Contest{ID: contest.ID, Title: contest.Title}
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
			resp.Contests = append(resp.Contests, makeContest(contest))
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
	return c.JSON(http.StatusOK, makeContest(contest))
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
	return c.JSON(http.StatusCreated, makeContest(contest))
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
	return c.JSON(http.StatusOK, makeContest(contest))
}

func (v *View) observeContestProblem(c echo.Context) error {
	return fmt.Errorf("not implemented")
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

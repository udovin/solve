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

// registerProblemHandlers registers handlers for problem management.
func (v *View) registerProblemHandlers(g *echo.Group) {
	g.GET(
		"/problems", v.observeProblems,
		v.sessionAuth,
		v.requireAuthRole(models.ObserveProblemsRole),
	)
	g.POST(
		"/problems", v.createProblem,
		v.sessionAuth,
		v.requireAuthRole(models.CreateProblemRole),
	)
	g.GET(
		"/problems/:problem", v.observeProblem,
		v.sessionAuth, v.extractProblem, v.extractProblemRoles,
		v.requireAuthRole(models.ObserveProblemRole),
	)
	g.DELETE(
		"/problems/:problem", v.deleteProblem,
		v.sessionAuth, v.requireAuth, v.extractProblem, v.extractProblemRoles,
		v.requireAuthRole(models.DeleteProblemRole),
	)
}

type Problem struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type problemSorter []Problem

func (v problemSorter) Len() int {
	return len(v)
}

func (v problemSorter) Less(i, j int) bool {
	return v[i].ID > v[j].ID
}

func (v problemSorter) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

type Problems struct {
	Problems []Problem `json:"problems"`
}

func (v *View) observeProblems(c echo.Context) error {
	roles, ok := c.Get(authRolesKey).(core.RoleSet)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	var resp Problems
	problems, err := v.core.Problems.All()
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	for _, problem := range problems {
		problemRoles := v.extendProblemRoles(c, roles, problem)
		if ok, err := v.core.HasRole(problemRoles, models.ObserveProblemRole); ok && err == nil {
			resp.Problems = append(resp.Problems, Problem{
				ID:    problem.ID,
				Title: problem.Title,
			})
		}
	}
	sort.Sort(problemSorter(resp.Problems))
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeProblem(c echo.Context) error {
	problem, ok := c.Get(problemKey).(models.Problem)
	if !ok {
		c.Logger().Error("problem not extracted")
		return fmt.Errorf("problem not extracted")
	}
	resp := Problem{
		ID:    problem.ID,
		Title: problem.Title,
	}
	return c.JSON(http.StatusOK, resp)
}

type createProblemForm struct {
	Title string `json:"title"`
}

func (f createProblemForm) validate() *errorResp {
	errors := errorFields{}
	if len(f.Title) < 4 {
		errors["title"] = errorField{Message: "title is too short"}
	}
	if len(f.Title) > 64 {
		errors["title"] = errorField{Message: "title is too long"}
	}
	if len(errors) > 0 {
		return &errorResp{
			Message:       "form has invalid fields",
			InvalidFields: errors,
		}
	}
	return nil
}

func (f createProblemForm) Update(problem *models.Problem) *errorResp {
	if err := f.validate(); err != nil {
		return err
	}
	problem.Title = f.Title
	return nil
}

func (v *View) createProblem(c echo.Context) error {
	var form createProblemForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var problem models.Problem
	if err := form.Update(&problem); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	if account, ok := c.Get(authAccountKey).(models.Account); ok {
		problem.OwnerID = models.NInt64(account.ID)
	}
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		return v.core.Problems.CreateTx(tx, &problem)
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusCreated, Problem{
		ID:    problem.ID,
		Title: problem.Title,
	})
}

func (v *View) deleteProblem(c echo.Context) error {
	problem, ok := c.Get(problemKey).(models.Problem)
	if !ok {
		c.Logger().Error("problem not extracted")
		return fmt.Errorf("problem not extracted")
	}
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		return v.core.Problems.DeleteTx(tx, problem.ID)
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, Problem{
		ID:    problem.ID,
		Title: problem.Title,
	})
}

func (v *View) extractProblem(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("problem"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return err
		}
		problem, err := v.core.Problems.Get(id)
		if err != nil {
			if err == sql.ErrNoRows {
				resp := errorResp{Message: "problem not found"}
				return c.JSON(http.StatusNotFound, resp)
			}
			c.Logger().Error(err)
			return err
		}
		c.Set(problemKey, problem)
		return next(c)
	}
}

func (v *View) extendProblemRoles(
	c echo.Context, roles core.RoleSet, problem models.Problem,
) core.RoleSet {
	problemRoles := roles.Clone()
	addRole := func(code string) {
		if err := v.core.AddRole(problemRoles, code); err != nil {
			c.Logger().Error(err)
		}
	}
	account, ok := c.Get(authAccountKey).(models.Account)
	if ok && problem.OwnerID != 0 && account.ID == int64(problem.OwnerID) {
		addRole(models.ObserveProblemRole)
		addRole(models.UpdateProblemRole)
		addRole(models.DeleteProblemRole)
	}
	return problemRoles
}

func (v *View) extractProblemRoles(next echo.HandlerFunc) echo.HandlerFunc {
	nextWrap := func(c echo.Context) error {
		problem, ok := c.Get(problemKey).(models.Problem)
		if !ok {
			c.Logger().Error("session not extracted")
			return fmt.Errorf("session not extracted")
		}
		roles, ok := c.Get(authRolesKey).(core.RoleSet)
		if !ok {
			c.Logger().Error("roles not extracted")
			return fmt.Errorf("roles not extracted")
		}
		c.Set(authRolesKey, v.extendProblemRoles(c, roles, problem))
		return next(c)
	}
	return v.extractAuthRoles(nextWrap)
}

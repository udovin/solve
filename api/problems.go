package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/labstack/echo"

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
	ID int64 `json:"id"`
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
				ID: problem.ID,
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
		ID: problem.ID,
	}
	return c.JSON(http.StatusOK, resp)
}

type createProblemForm struct {
}

func (f createProblemForm) validate() *errorResp {
	return nil
}

func (f createProblemForm) Update(problem *models.Problem) *errorResp {
	if err := f.validate(); err != nil {
		return err
	}
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
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		return v.core.Problems.CreateTx(tx, &problem)
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusCreated, Problem{
		ID: problem.ID,
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
		ID: problem.ID,
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
	authUser, ok := c.Get(authUserKey).(models.User)
	if ok && problem.OwnerID != 0 && authUser.ID == int64(problem.OwnerID) {
		addRole(models.UpdateProblemRole)
		addRole(models.DeleteProblemRole)
	}
	addRole(models.ObserveProblemRole)
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

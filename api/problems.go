package api

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

// registerProblemHandlers registers handlers for problem management.
func (v *View) registerProblemHandlers(g *echo.Group) {
	g.GET(
		"/v0/problems", v.observeProblems,
		v.sessionAuth,
		v.requireAuthRole(models.ObserveProblemsRole),
	)
	if v.core.Config.Storage != nil {
		g.POST(
			"/v0/problems", v.createProblem,
			v.sessionAuth, v.requireAuth,
			v.requireAuthRole(models.CreateProblemRole),
		)
	}
	g.GET(
		"/v0/problems/:problem", v.observeProblem,
		v.sessionAuth, v.extractProblem, v.extractProblemRoles,
		v.requireAuthRole(models.ObserveProblemRole),
	)
	g.DELETE(
		"/v0/problems/:problem", v.deleteProblem,
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

func makeProblem(problem models.Problem) Problem {
	return Problem{
		ID:    problem.ID,
		Title: problem.Title,
	}
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
			resp.Problems = append(resp.Problems, makeProblem(problem))
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
	return c.JSON(http.StatusOK, makeProblem(problem))
}

type createProblemForm struct {
	Title string `form:"title"`
}

func (f createProblemForm) validate() *errorResponse {
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

func (f createProblemForm) Update(problem *models.Problem) *errorResponse {
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
		if err := v.core.Problems.CreateTx(tx, &problem); err != nil {
			return err
		}
		dst, err := os.Create(filepath.Join(
			v.core.Config.Storage.ProblemsDir,
			fmt.Sprintf("%d.zip", problem.ID),
		))
		if err != nil {
			return err
		}
		defer dst.Close()
		_, err = io.Copy(dst, src)
		return err
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusCreated, makeProblem(problem))
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
	return c.JSON(http.StatusOK, makeProblem(problem))
}

func (v *View) extractProblem(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("problem"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return err
		}
		problem, err := v.core.Problems.Get(id)
		if err == sql.ErrNoRows {
			if err := v.core.Problems.SyncTx(v.core.DB); err != nil {
				return err
			}
			problem, err = v.core.Problems.Get(id)
		}
		if err != nil {
			if err == sql.ErrNoRows {
				resp := errorResponse{Message: "problem not found"}
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
	addRole := func(name string) {
		if err := v.core.AddRole(problemRoles, name); err != nil {
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

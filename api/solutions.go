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

// registerSolutionHandlers registers handlers for solution management.
func (v *View) registerSolutionHandlers(g *echo.Group) {
	if v.core.Users == nil {
		return
	}
	g.GET(
		"/v0/solutions", v.observeSolutions,
		v.sessionAuth,
		v.requireAuthRole(models.ObserveSolutionsRole),
	)
	g.GET(
		"/v0/solutions/:solution", v.observeSolution,
		v.sessionAuth, v.extractSolution, v.extractSolutionRoles,
		v.requireAuthRole(models.ObserveSolutionRole),
	)
}

type Solution struct {
	ID         int64           `json:"id"`
	Problem    *Problem        `json:"problem"`
	User       *User           `json:"user"`
	Report     *SolutionReport `json:"report"`
	CreateTime int64           `json:"create_time"`
}

type Solutions struct {
	Solutions []Solution `json:"solutions"`
}

type solutionSorter []Solution

func (v solutionSorter) Len() int {
	return len(v)
}

func (v solutionSorter) Less(i, j int) bool {
	return v[i].ID > v[j].ID
}

func (v solutionSorter) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func makeSolution(c echo.Context, solution models.Solution, roles core.RoleSet, core *core.Core) Solution {
	resp := Solution{
		ID:         solution.ID,
		CreateTime: solution.CreateTime,
	}
	if problem, err := core.Problems.Get(solution.ProblemID); err == nil {
		problemResp := makeProblem(problem)
		resp.Problem = &problemResp
	}
	if account, err := core.Accounts.Get(solution.AuthorID); err == nil {
		switch account.Kind {
		case models.UserAccount:
			if user, err := core.Users.GetByAccount(account.ID); err == nil {
				userResp := makeUser(c, user, roles, core)
				resp.User = &userResp
			}
		}
	}
	if report, err := solution.GetReport(); err == nil {
		reportResp := SolutionReport{
			Verdict:    report.Verdict.String(),
			CompileLog: report.CompileLog,
		}
		for _, test := range report.Tests {
			reportResp.Tests = append(reportResp.Tests, TestReport{
				Verdict:  test.Verdict.String(),
				CheckLog: test.CheckLog,
			})
		}
		resp.Report = &reportResp
	}
	return resp
}

func (v *View) observeSolutions(c echo.Context) error {
	roles, ok := c.Get(authRolesKey).(core.RoleSet)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	var resp Solutions
	solutions, err := v.core.Solutions.All()
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	for _, solution := range solutions {
		solutionRoles := v.extendSolutionRoles(c, roles, solution)
		if ok, err := v.core.HasRole(
			solutionRoles, models.ObserveSolutionRole,
		); ok && err == nil {
			resp.Solutions = append(
				resp.Solutions,
				makeSolution(c, solution, solutionRoles, v.core),
			)
		}
	}
	sort.Sort(solutionSorter(resp.Solutions))
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeSolution(c echo.Context) error {
	solution, ok := c.Get(solutionKey).(models.Solution)
	if !ok {
		c.Logger().Error("solution not extracted")
		return fmt.Errorf("solution not extracted")
	}
	roles, ok := c.Get(authRolesKey).(core.RoleSet)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	return c.JSON(http.StatusOK, makeSolution(c, solution, roles, v.core))
}

func (v *View) extractSolution(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("solution"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			resp := errorResponse{Message: "invalid solution ID"}
			return c.JSON(http.StatusBadRequest, resp)
		}
		solution, err := v.core.Solutions.Get(id)
		if err == sql.ErrNoRows {
			if err := v.core.Solutions.SyncTx(v.core.DB); err != nil {
				return err
			}
			solution, err = v.core.Solutions.Get(id)
		}
		if err != nil {
			if err == sql.ErrNoRows {
				resp := errorResponse{Message: "solution not found"}
				return c.JSON(http.StatusNotFound, resp)
			}
			c.Logger().Error(err)
			return err
		}
		c.Set(solutionKey, solution)
		return next(c)
	}
}

func (v *View) extendSolutionRoles(
	c echo.Context, roles core.RoleSet, solution models.Solution,
) core.RoleSet {
	solutionRoles := roles.Clone()
	if solution.ID == 0 {
		return solutionRoles
	}
	addRole := func(code string) {
		if err := v.core.AddRole(solutionRoles, code); err != nil {
			c.Logger().Error(err)
		}
	}
	account, ok := c.Get(authAccountKey).(models.Account)
	if ok {
		if solution.AuthorID != 0 && account.ID == int64(solution.AuthorID) {
			addRole(models.ObserveSolutionRole)
		}
	}
	return solutionRoles
}

func (v *View) extractSolutionRoles(next echo.HandlerFunc) echo.HandlerFunc {
	nextWrap := func(c echo.Context) error {
		solution, ok := c.Get(solutionKey).(models.Solution)
		if !ok {
			c.Logger().Error("contest not extracted")
			return fmt.Errorf("contest not extracted")
		}
		roles, ok := c.Get(authRolesKey).(core.RoleSet)
		if !ok {
			c.Logger().Error("roles not extracted")
			return fmt.Errorf("roles not extracted")
		}
		c.Set(authRolesKey, v.extendSolutionRoles(c, roles, solution))
		return next(c)
	}
	return v.extractAuthRoles(nextWrap)
}

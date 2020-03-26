package api

import (
	"database/sql"
	"net/http"
	"sort"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

type Solution struct {
	models.Solution
	User    *models.User   `json:""`
	Problem *Problem       `json:""`
	Report  *models.Report `json:""`
}

func (v *View) GetSolution(c echo.Context) error {
	solutionID, err := strconv.ParseInt(c.Param("SolutionID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	solution, err := v.buildSolution(solutionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	user, ok := c.Get(authUserKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if !v.canGetSolution(user, solution.Solution) {
		return c.NoContent(http.StatusForbidden)
	}
	return c.JSON(http.StatusOK, solution)
}

func (v *View) RejudgeSolution(c echo.Context) error {
	solutionID, err := strconv.ParseInt(c.Param("SolutionID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	solution, err := v.buildSolution(solutionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	_, ok := c.Get(authUserKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	report := models.Report{
		SolutionID: solution.ID,
	}
	if err := v.core.Reports.Create(&report); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, report)
}

func (v *View) GetSolutions(c echo.Context) error {
	_, ok := c.Get(authUserKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	var solutions []Solution
	list, err := v.core.Solutions.All()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	for _, m := range list {
		if solution, err := v.buildSolution(m.ID); err == nil {
			solution.SourceCode = ""
			if solution.Report != nil {
				solution.Report.Data.PrecompileLogs = models.ReportDataLogs{}
				solution.Report.Data.CompileLogs = models.ReportDataLogs{}
				solution.Report.Data.Tests = nil
			}
			solutions = append(solutions, solution)
		}
	}
	sort.Sort(solutionSorter(solutions))
	return c.JSON(http.StatusOK, solutions)
}

type reportDiff struct {
	Points  *float64 `json:""`
	Defense *int8    `json:""`
}

func (v *View) createSolutionReport(c echo.Context) error {
	solutionID, err := strconv.ParseInt(c.Param("SolutionID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var diff reportDiff
	if err := c.Bind(&diff); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	solution, err := v.buildSolution(solutionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	_, ok := c.Get(authUserKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	report, err := v.core.Reports.GetLatest(solution.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if diff.Defense != nil {
		report.Data.Defense = diff.Defense
	}
	if diff.Points != nil {
		report.Data.Points = diff.Points
	}
	if err := v.core.Reports.Create(&report); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, report)
}

func (v *View) canGetSolution(
	user models.User, solution models.Solution,
) bool {
	if user.ID == solution.UserID {
		return true
	}
	if solution.ContestID > 0 {
		contest, err := v.core.Contests.Get(solution.ContestID)
		if err == nil && user.ID == contest.UserID {
			return true
		}
	}
	return false
}

func (v *View) buildSolution(id int64) (Solution, error) {
	solution, err := v.core.Solutions.Get(id)
	if err != nil {
		return Solution{}, err
	}
	result := Solution{
		Solution: solution,
	}
	if user, err := v.core.Users.Get(solution.UserID); err == nil {
		result.User = &user
	}
	if problem, err := v.buildProblem(solution.ProblemID); err == nil {
		problem.Description = ""
		result.Problem = &problem
	}
	if report, err := v.core.Reports.GetLatest(solution.ID); err == nil {
		result.Report = &report
	}
	return result, nil
}

type solutionSorter []Solution

func (c solutionSorter) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c solutionSorter) Len() int {
	return len(c)
}

func (c solutionSorter) Less(i, j int) bool {
	return c[i].ID > c[j].ID
}

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

func (s *Server) GetSolution(c echo.Context) error {
	solutionID, err := strconv.ParseInt(c.Param("SolutionID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	solution, err := s.buildSolution(solutionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if !s.canGetSolution(user, solution.Solution) {
		return c.NoContent(http.StatusForbidden)
	}
	return c.JSON(http.StatusOK, solution)
}

func (s *Server) RejudgeSolution(c echo.Context) error {
	solutionID, err := strconv.ParseInt(c.Param("SolutionID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	solution, err := s.buildSolution(solutionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if !user.IsSuper {
		return c.NoContent(http.StatusForbidden)
	}
	report := models.Report{
		SolutionID: solution.ID,
	}
	if err := s.app.Reports.Create(&report); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, report)
}

func (s *Server) GetSolutions(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if !user.IsSuper {
		return c.NoContent(http.StatusForbidden)
	}
	var solutions []Solution
	list, err := s.app.Solutions.All()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	for _, m := range list {
		if solution, err := s.buildSolution(m.ID); err == nil {
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

func (s *Server) createSolutionReport(c echo.Context) error {
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
	solution, err := s.buildSolution(solutionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if !user.IsSuper {
		return c.NoContent(http.StatusForbidden)
	}
	report, err := s.app.Reports.GetLatest(solution.ID)
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
	if err := s.app.Reports.Create(&report); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, report)
}

func (s *Server) canGetSolution(
	user models.User, solution models.Solution,
) bool {
	if user.IsSuper {
		return true
	}
	if user.ID == solution.UserID {
		return true
	}
	if solution.ContestID > 0 {
		contest, err := s.app.Contests.Get(solution.ContestID)
		if err == nil && user.ID == contest.UserID {
			return true
		}
	}
	return false
}

func (s *Server) buildSolution(id int64) (Solution, error) {
	solution, err := s.app.Solutions.Get(id)
	if err != nil {
		return Solution{}, err
	}
	result := Solution{
		Solution: solution,
	}
	if user, err := s.app.Users.Get(solution.UserID); err == nil {
		result.User = &user
	}
	if problem, err := s.buildProblem(solution.ProblemID); err == nil {
		problem.Description = ""
		result.Problem = &problem
	}
	if report, err := s.app.Reports.GetLatest(solution.ID); err == nil {
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

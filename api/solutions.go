package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

type Solution struct {
	models.Solution
	Report *models.Report `json:""`
}

func (v *View) GetSolution(c echo.Context) error {
	solutionID, err := strconv.ParseInt(c.Param("SolutionID"), 10, 64)
	if err != nil {
		return err
	}
	solution, ok := v.buildSolution(solutionID)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if !v.canGetSolution(user, solution.Solution) {
		return c.NoContent(http.StatusForbidden)
	}
	return c.JSON(http.StatusOK, solution)
}

func (v *View) canGetSolution(
	user models.User, solution models.Solution,
) bool {
	if user.IsSuper {
		return true
	}
	if user.ID == solution.UserID {
		return true
	}
	if solution.ContestID > 0 {
		contest, ok := v.app.Contests.Get(solution.ContestID)
		if ok && user.ID == contest.UserID {
			return true
		}
	}
	return false
}

func (v *View) buildSolution(id int64) (Solution, bool) {
	solution, ok := v.app.Solutions.Get(id)
	if !ok {
		return Solution{}, false
	}
	report, ok := v.app.Reports.GetLatest(solution.ID)
	if ok {
		return Solution{
			Solution: solution,
			Report:   &report,
		}, true
	}
	return Solution{
		Solution: solution,
	}, true
}

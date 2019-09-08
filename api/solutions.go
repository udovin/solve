package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

func (v *View) GetSolution(c echo.Context) error {
	solutionID, err := strconv.ParseInt(c.Param("SolutionID"), 10, 64)
	if err != nil {
		return err
	}
	solution, ok := v.app.Sessions.Get(solutionID)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if solution.UserID != user.ID {
		return c.NoContent(http.StatusForbidden)
	}
	return c.JSON(http.StatusOK, solution)
}

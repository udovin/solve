package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

func (v *View) CreateSolution(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	var solution models.Solution
	if err := c.Bind(&solution); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	solution.UserID = user.ID
	if err := v.app.Solutions.Create(&solution); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, solution)
}

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

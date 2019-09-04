package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

func (v *View) CreateProblem(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

type Problem struct {
	models.Problem
	Title       string `json:""`
	Description string `json:""`
}

func (v *View) GetProblem(c echo.Context) error {
	problemID, err := strconv.ParseInt(c.Param("ProblemID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	problem, ok := v.buildProblem(problemID)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	return c.JSON(http.StatusOK, problem)
}

func (v *View) buildProblem(id int64) (Problem, bool) {
	problem, ok := v.app.Problems.Get(id)
	if !ok {
		return Problem{}, false
	}
	statement, ok := v.app.Statements.GetByProblem(problem.ID)
	if !ok {
		return Problem{}, false
	}
	return Problem{
		Problem:     problem,
		Title:       statement.Title,
		Description: statement.Description,
	}, true
}

func (v *View) UpdateProblem(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) DeleteProblem(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

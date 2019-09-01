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

type ProblemResponse struct {
	models.Problem
	Statement models.Statement `json:""`
}

func (v *View) GetProblem(c echo.Context) error {
	problemID, err := strconv.ParseInt(c.Param("ProblemID"), 10, 64)
	if err != nil {
		return err
	}
	problem, ok := v.app.Problems.Get(problemID)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	statement, ok := v.app.Statements.GetByProblem(problem.ID)
	if !ok {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, ProblemResponse{
		Problem:   problem,
		Statement: statement,
	})
}

func (v *View) UpdateProblem(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) DeleteProblem(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

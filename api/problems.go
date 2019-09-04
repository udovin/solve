package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

type Problem struct {
	models.Problem
	Title       string `json:""`
	Description string `json:""`
}

func (v *View) CreateProblem(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	var problem Problem
	if err := c.Bind(&problem); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	problem.UserID = user.ID
	tx, err := v.app.Problems.Manager.Begin()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := v.app.Problems.CreateTx(tx, &problem.Problem); err != nil {
		if err := tx.Rollback(); err != nil {
			c.Logger().Error(err)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	statement := models.Statement{
		ProblemID:   problem.ID,
		Title:       problem.Title,
		Description: problem.Description,
	}
	if err := v.app.Statements.CreateTx(tx, &statement); err != nil {
		if err := tx.Rollback(); err != nil {
			c.Logger().Error(err)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := tx.Commit(); err != nil {
		if err := tx.Rollback(); err != nil {
			c.Logger().Error(err)
		}
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, problem)
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

package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo"
)

func (v *View) CreateProblem(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) GetProblem(c echo.Context) error {
	problemID, err := strconv.ParseInt(c.Param("ProblemID"), 10, 60)
	if err != nil {
		return err
	}
	problem, ok := v.app.ProblemStore.Get(problemID)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	return c.JSON(http.StatusOK, problem)
}

func (v *View) UpdateProblem(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) DeleteProblem(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

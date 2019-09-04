package api

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

func (v *View) GetContestList(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) CreateContest(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

type Contest struct {
	models.Contest
	Problems []ContestProblem `json:""`
}

type ContestProblem struct {
	Problem
	Code string `json:""`
}

type ContestProblemSorter []ContestProblem

func (c ContestProblemSorter) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c ContestProblemSorter) Len() int {
	return len(c)
}

func (c ContestProblemSorter) Less(i, j int) bool {
	return c[i].Code < c[j].Code
}

func (v *View) GetContest(c echo.Context) error {
	contestID, err := strconv.ParseInt(c.Param("ContestID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	contest, ok := v.buildContest(contestID)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	return c.JSON(http.StatusOK, contest)
}

func (v *View) buildContest(id int64) (Contest, bool) {
	contest, ok := v.app.Contests.Get(id)
	if !ok {
		return Contest{}, false
	}
	result := Contest{
		Contest: contest,
	}
	for _, contestProblem := range v.app.ContestProblems.GetByContest(id) {
		problem, ok := v.buildProblem(contestProblem.ProblemID)
		if !ok {
			continue
		}
		result.Problems = append(result.Problems, ContestProblem{
			Problem: problem,
			Code:    contestProblem.Code,
		})
	}
	sort.Sort(ContestProblemSorter(result.Problems))
	return result, true
}

func (v *View) UpdateContest(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) DeleteContest(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

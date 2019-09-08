package api

import (
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

type Contest struct {
	models.Contest
	Problems []ContestProblem `json:""`
}

type ContestProblem struct {
	Problem
	Code string `json:""`
}

func (v *View) GetContests(c echo.Context) error {
	contests := v.app.Contests.All()
	if contests == nil {
		contests = make([]models.Contest, 0)
	}
	return c.JSON(http.StatusOK, contests)
}

func (v *View) CreateContest(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	var contest models.Contest
	if err := c.Bind(&contest); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	contest.UserID = user.ID
	if err := v.app.Contests.Create(&contest); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, contest)
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

func (v *View) GetContestProblem(c echo.Context) error {
	contestID, err := strconv.ParseInt(c.Param("ContestID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	problemCode := c.Param("ProblemCode")
	var contestProblem models.ContestProblem
	for _, problem := range v.app.ContestProblems.GetByContest(contestID) {
		if problem.Code == problemCode {
			contestProblem = problem
			break
		}
	}
	if contestProblem.Code != problemCode {
		return c.NoContent(http.StatusNotFound)
	}
	problem, ok := v.buildProblem(contestProblem.ProblemID)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	return c.JSON(http.StatusOK, problem)
}

func (v *View) CreateContestProblem(c echo.Context) error {
	contestID, err := strconv.ParseInt(c.Param("ContestID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var contestProblem models.ContestProblem
	if err := c.Bind(&contestProblem); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	contestProblem.ContestID = contestID
	log.Println(contestProblem)
	if _, ok := v.app.Contests.Get(contestProblem.ContestID); !ok {
		return c.NoContent(http.StatusNotFound)
	}
	if _, ok := v.app.Problems.Get(contestProblem.ProblemID); !ok {
		return c.NoContent(http.StatusNotFound)
	}
	if err := v.app.ContestProblems.Create(&contestProblem); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, contestProblem)
}

func (v *View) CreateContestSolution(c echo.Context) error {
	contestID, err := strconv.ParseInt(c.Param("ContestID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	problemCode := c.Param("ProblemCode")
	var contestProblem models.ContestProblem
	for _, problem := range v.app.ContestProblems.GetByContest(contestID) {
		if problem.Code == problemCode {
			contestProblem = problem
			break
		}
	}
	if contestProblem.Code != problemCode {
		return c.NoContent(http.StatusNotFound)
	}
	var solution models.Solution
	if err := c.Bind(&solution); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	if _, ok := v.app.Compilers.Get(solution.CompilerID); !ok {
		return c.NoContent(http.StatusNotFound)
	}
	solution.UserID = user.ID
	solution.ContestID = contestProblem.ContestID
	solution.ProblemID = contestProblem.ProblemID
	tx, err := v.app.Solutions.Manager.Begin()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := v.app.Solutions.CreateTx(tx, &solution); err != nil {
		c.Logger().Error(err)
		if err := tx.Rollback(); err != nil {
			c.Logger().Error(err)
		}
		return c.NoContent(http.StatusInternalServerError)
	}
	report := models.Report{
		SolutionID: solution.ID,
	}
	if err := v.app.Reports.CreateTx(tx, &report); err != nil {
		c.Logger().Error(err)
		if err := tx.Rollback(); err != nil {
			c.Logger().Error(err)
		}
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := tx.Commit(); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, solution)
}

func (v *View) UpdateContest(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) DeleteContest(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

type contestProblemSorter []ContestProblem

func (c contestProblemSorter) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c contestProblemSorter) Len() int {
	return len(c)
}

func (c contestProblemSorter) Less(i, j int) bool {
	return c[i].Code < c[j].Code
}

func (v *View) buildContest(id int64) (Contest, bool) {
	contest, ok := v.app.Contests.Get(id)
	if !ok {
		return Contest{}, false
	}
	result := Contest{
		Contest:  contest,
		Problems: make([]ContestProblem, 0),
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
	sort.Sort(contestProblemSorter(result.Problems))
	return result, true
}

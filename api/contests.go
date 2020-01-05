package api

import (
	"database/sql"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

type Contest struct {
	models.Contest
	Problems []ContestProblem `json:""`
}

type ContestProblem struct {
	Problem
	ContestID int64  `json:""`
	Code      string `json:""`
}

func (s *Server) GetContests(c echo.Context) error {
	contests, err := s.app.Contests.All()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if contests == nil {
		contests = make([]models.Contest, 0)
	}
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	var result []models.Contest
	for _, contest := range contests {
		if s.canGetContest(user, contest) {
			result = append(result, contest)
		}
	}
	sort.Sort(contestModelSorter(result))
	return c.JSON(http.StatusOK, result)
}

func (s *Server) CreateContest(c echo.Context) error {
	var contest models.Contest
	if err := c.Bind(&contest); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if !user.IsSuper {
		return c.NoContent(http.StatusForbidden)
	}
	contest.UserID = user.ID
	if err := s.app.Contests.Create(&contest); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, contest)
}

func (s *Server) GetContest(c echo.Context) error {
	contestID, err := strconv.ParseInt(c.Param("ContestID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	contest, err := s.buildContest(contestID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if !s.canGetContest(user, contest.Contest) {
		return c.NoContent(http.StatusForbidden)
	}
	return c.JSON(http.StatusOK, contest)
}

func (s *Server) GetContestSolutions(c echo.Context) error {
	contestID, err := strconv.ParseInt(c.Param("ContestID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	contest, err := s.app.Contests.Get(contestID)
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
	if !s.canGetContest(user, contest) {
		return c.NoContent(http.StatusForbidden)
	}
	var result []Solution
	solutions, err := s.app.Solutions.GetByContest(contest.ID)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	for _, model := range solutions {
		if s.canGetSolution(user, model) {
			if solution, err := s.buildSolution(model.ID); err == nil {
				solution.SourceCode = ""
				if solution.Report != nil {
					solution.Report.Data.PrecompileLogs = models.ReportDataLogs{}
					solution.Report.Data.CompileLogs = models.ReportDataLogs{}
					solution.Report.Data.Tests = nil
				}
				result = append(result, solution)
			}
		}
	}
	sort.Sort(solutionSorter(result))
	return c.JSON(http.StatusOK, result)
}

func (s *Server) GetContestProblem(c echo.Context) error {
	contestID, err := strconv.ParseInt(c.Param("ContestID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	problemCode := c.Param("ProblemCode")
	var contestProblem models.ContestProblem
	problems, err := s.app.ContestProblems.GetByContest(contestID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	for _, problem := range problems {
		if problem.Code == problemCode {
			contestProblem = problem
			break
		}
	}
	if contestProblem.Code != problemCode {
		return c.NoContent(http.StatusNotFound)
	}
	problem, err := s.buildProblem(contestProblem.ProblemID)
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
	solutions, err := s.app.Solutions.GetByProblemUser(problem.ID, user.ID)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	for _, model := range solutions {
		solution, err := s.buildSolution(model.ID)
		if err == nil && solution.ContestID == contestID {
			solution.SourceCode = ""
			if solution.Report != nil {
				solution.Report.Data.PrecompileLogs = models.ReportDataLogs{}
				solution.Report.Data.CompileLogs = models.ReportDataLogs{}
				solution.Report.Data.Tests = nil
			}
			problem.Solutions = append(problem.Solutions, solution)
		}
	}
	sort.Sort(solutionSorter(problem.Solutions))
	return c.JSON(http.StatusOK, problem)
}

func (s *Server) CreateContestProblem(c echo.Context) error {
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
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if !user.IsSuper {
		return c.NoContent(http.StatusForbidden)
	}
	if _, err := s.app.Contests.Get(contestProblem.ContestID); err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if _, err := s.app.Problems.Get(contestProblem.ProblemID); err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := s.app.ContestProblems.Create(&contestProblem); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, contestProblem)
}

func (s *Server) CreateContestSolution(c echo.Context) error {
	contestID, err := strconv.ParseInt(c.Param("ContestID"), 10, 64)
	if err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	contest, err := s.app.Contests.Get(contestID)
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
	if !s.canGetContest(user, contest) {
		return c.NoContent(http.StatusForbidden)
	}
	if !s.canCreateSolution(user, contest) {
		return c.NoContent(http.StatusForbidden)
	}
	problemCode := c.Param("ProblemCode")
	var contestProblem models.ContestProblem
	problems, err := s.app.ContestProblems.GetByContest(contestID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	for _, problem := range problems {
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
	if _, err := s.app.Compilers.Get(solution.CompilerID); err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	solution.UserID = user.ID
	solution.ContestID = contestProblem.ContestID
	solution.ProblemID = contestProblem.ProblemID
	tx, err := s.app.Solutions.Manager.Begin()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := s.app.Solutions.CreateTx(tx, &solution); err != nil {
		c.Logger().Error(err)
		if err := tx.Rollback(); err != nil {
			c.Logger().Error(err)
		}
		return c.NoContent(http.StatusInternalServerError)
	}
	report := models.Report{
		SolutionID: solution.ID,
	}
	if err := s.app.Reports.CreateTx(tx, &report); err != nil {
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

func (s *Server) UpdateContest(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (s *Server) DeleteContest(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (s *Server) buildContest(id int64) (Contest, error) {
	contest, err := s.app.Contests.Get(id)
	if err != nil {
		return Contest{}, err
	}
	result := Contest{
		Contest:  contest,
		Problems: make([]ContestProblem, 0),
	}
	problems, err := s.app.ContestProblems.GetByContest(id)
	if err != nil {
		return Contest{}, err
	}
	for _, contestProblem := range problems {
		problem, err := s.buildProblem(contestProblem.ProblemID)
		if err != nil {
			continue
		}
		problem.Description = ""
		result.Problems = append(result.Problems, ContestProblem{
			Problem:   problem,
			ContestID: contestProblem.ContestID,
			Code:      contestProblem.Code,
		})
	}
	sort.Sort(contestProblemSorter(result.Problems))
	return result, nil
}

func (s *Server) canGetContest(
	user models.User, contest models.Contest,
) bool {
	if user.IsSuper {
		return true
	}
	if contest.Config.BeginTime != nil {
		if time.Now().Unix() < *contest.Config.BeginTime {
			return false
		}
	}
	if user.ID == contest.UserID {
		return true
	}
	participants, err := s.app.Participants.GetByContestUser(
		contest.ID, user.ID,
	)
	return err == nil && len(participants) > 0
}

func (s *Server) canCreateSolution(
	user models.User, contest models.Contest,
) bool {
	if user.IsSuper {
		return true
	}
	if contest.Config.EndTime != nil {
		if time.Now().Unix() >= *contest.Config.EndTime {
			return false
		}
	}
	return true
}

type contestModelSorter []models.Contest

func (c contestModelSorter) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c contestModelSorter) Len() int {
	return len(c)
}

func (c contestModelSorter) Less(i, j int) bool {
	return c[i].ID > c[j].ID
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

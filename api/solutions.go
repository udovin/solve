package api

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

// registerSolutionHandlers registers handlers for solution management.
func (v *View) registerSolutionHandlers(g *echo.Group) {
	g.GET(
		"/v0/solutions", v.observeSolutions,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(models.ObserveSolutionsRole),
	)
	g.GET(
		"/v0/solutions/:solution", v.observeSolution,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractSolution,
		v.requirePermission(models.ObserveSolutionRole),
	)
}

type Solution struct {
	ID         int64           `json:"id"`
	Problem    *Problem        `json:"problem"`
	Compiler   *Compiler       `json:"compiler"`
	User       *User           `json:"user,omitempty"`
	ScopeUser  *ScopeUser      `json:"scope_user,omitempty"`
	Content    string          `json:"content,omitempty"`
	Report     *SolutionReport `json:"report"`
	CreateTime int64           `json:"create_time"`
}

type Solutions struct {
	Solutions   []Solution `json:"solutions"`
	NextBeginID int64      `json:"next_begin_id,omitempty"`
}

func (v *View) tryFindSolutionTask(id int64) (models.Task, error) {
	tasks, err := v.core.Tasks.FindBySolution(id)
	if err != nil {
		return models.Task{}, err
	}
	var lastTask models.Task
	for _, task := range tasks {
		if task.Kind == models.JudgeSolutionTask {
			var config models.JudgeSolutionTaskConfig
			if err := task.ScanConfig(&config); err != nil {
				continue
			}
			if task.ID > lastTask.ID {
				lastTask = task
			}
		}
	}
	if lastTask.ID == 0 {
		return models.Task{}, sql.ErrNoRows
	}
	return lastTask, nil
}

func (v *View) findSolutionTask(c echo.Context, id int64) (models.Task, error) {
	task, err := v.tryFindSolutionTask(id)
	if err == sql.ErrNoRows {
		if err := v.core.Tasks.Sync(getContext(c)); err != nil {
			return models.Task{}, err
		}
		return v.tryFindSolutionTask(id)
	}
	return task, err
}

func (v *View) makeSolutionContent(c echo.Context, solution models.Solution) string {
	var result string
	if solution.Content != "" {
		if s := string(solution.Content); utf8.ValidString(s) {
			result = s
		}
	} else if solution.ContentID != 0 {
		if file, err := v.files.DownloadFile(c.Request().Context(), int64(solution.ContentID)); err == nil {
			defer file.Close()
			var content bytes.Buffer
			if _, err := io.CopyN(&content, file, 64*1024); err == nil || err == io.EOF {
				if s := content.String(); utf8.ValidString(s) {
					result = s
				}
			}
		}
	}
	return result
}

type TestReport struct {
	Verdict    models.Verdict `json:"verdict"`
	UsedTime   int64          `json:"used_time,omitempty"`
	UsedMemory int64          `json:"used_memory,omitempty"`
	CheckLog   string         `json:"check_log,omitempty"`
	Input      string         `json:"input,omitempty"`
	Output     string         `json:"output,omitempty"`
}

type SolutionReport struct {
	Verdict    string       `json:"verdict"`
	Points     *float64     `json:"points,omitempty"`
	UsedTime   int64        `json:"used_time,omitempty"`
	UsedMemory int64        `json:"used_memory,omitempty"`
	Tests      []TestReport `json:"tests,omitempty"`
	TestNumber int          `json:"test_number,omitempty"`
	CompileLog string       `json:"compile_log,omitempty"`
}

func (v *View) makeSolutionReport(c echo.Context, solution models.Solution, withLogs bool) *SolutionReport {
	report, err := solution.GetReport()
	if err != nil {
		return &SolutionReport{
			Verdict: models.FailedTask.String(),
		}
	}
	if report == nil {
		task, err := v.findSolutionTask(c, solution.ID)
		if err != nil {
			return &SolutionReport{
				Verdict: models.FailedTask.String(),
			}
		}
		if task.Status == models.SucceededTask {
			return &SolutionReport{
				Verdict: models.RunningTask.String(),
			}
		}
		return &SolutionReport{
			Verdict: task.Status.String(),
		}
	}
	permissions, ok := c.Get(permissionCtxKey).(managers.Permissions)
	if !ok {
		permissions = managers.PermissionSet{}
	}
	resp := SolutionReport{
		Verdict:    report.Verdict.String(),
		Points:     report.Points,
		UsedTime:   report.Usage.Time,
		UsedMemory: report.Usage.Memory,
	}
	if report.Verdict != models.Accepted &&
		permissions.HasPermission(models.ObserveSolutionReportTestNumber) {
		for i, test := range report.Tests {
			if test.Verdict == report.Verdict {
				resp.TestNumber = i + 1
				break
			}
		}
	}
	if withLogs &&
		permissions.HasPermission(models.ObserveSolutionReportCheckerLogs) {
		resp.CompileLog = report.Compile.Log
		for _, test := range report.Tests {
			resp.Tests = append(resp.Tests, TestReport{
				Verdict:    test.Verdict,
				CheckLog:   test.Check.Log,
				UsedTime:   test.Usage.Time,
				UsedMemory: test.Usage.Memory,
			})
		}
	}
	return &resp
}

func (v *View) makeSolution(
	c echo.Context, ctx *managers.AccountContext, solution models.Solution, withLogs bool,
) Solution {
	resp := Solution{
		ID:         solution.ID,
		CreateTime: solution.CreateTime,
	}
	if problem, err := v.core.Problems.Get(getContext(c), solution.ProblemID); err == nil {
		problemResp := v.makeProblem(c, problem, managers.PermissionSet{}, false, false)
		resp.Problem = &problemResp
	}
	if compiler, err := v.core.Compilers.Get(getContext(c), solution.CompilerID); err == nil {
		compilerResp := makeCompiler(compiler)
		resp.Compiler = &compilerResp
	}
	if account, err := v.core.Accounts.Get(getContext(c), solution.AuthorID); err == nil {
		switch account.Kind {
		case models.UserAccount:
			if user, err := v.core.Users.GetByAccount(account.ID); err == nil {
				resp.User = &User{
					ID:    user.ID,
					Login: user.Login,
				}
			}
		case models.ScopeUserAccount:
			if user, err := v.core.ScopeUsers.GetByAccount(account.ID); err == nil {
				resp.ScopeUser = &ScopeUser{
					ID:    user.ID,
					Login: user.Login,
					Title: string(user.Title),
				}
			}
		}
	}
	if withLogs {
		resp.Content = v.makeSolutionContent(c, solution)
	}
	resp.Report = v.makeSolutionReport(c, solution, withLogs)
	return resp
}

type solutionsFilter struct {
	ProblemID int64          `query:"problem_id"`
	Verdict   models.Verdict `query:"verdict"`
	BeginID   int64          `query:"begin_id"`
	Limit     int            `query:"limit"`
}

func (f *solutionsFilter) Filter(solution models.Solution) bool {
	if f.ProblemID != 0 && solution.ProblemID != f.ProblemID {
		return false
	}
	if f.BeginID != 0 && solution.ID < f.BeginID {
		return false
	}
	if f.Verdict != 0 {
		report, err := solution.GetReport()
		if err != nil {
			return false
		}
		if report.Verdict != f.Verdict {
			return false
		}
	}
	return true
}

func (v *View) observeSolutions(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		c.Logger().Error("auth not extracted")
		return fmt.Errorf("auth not extracted")
	}
	filter := solutionsFilter{Limit: 200}
	if err := c.Bind(&filter); err != nil {
		c.Logger().Warn(err)
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Invalid filter."),
		}
	}
	var resp Solutions
	solutions, err := v.core.Solutions.ReverseAll(getContext(c), 0)
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	defer func() { _ = solutions.Close() }()
	for solutions.Next() {
		solution := solutions.Row()
		if !filter.Filter(solution) {
			continue
		}
		permissions := v.getSolutionPermissions(accountCtx, solution)
		if permissions.HasPermission(models.ObserveSolutionRole) {
			if len(resp.Solutions) > filter.Limit {
				resp.NextBeginID = solution.ID
				break
			}
			resp.Solutions = append(resp.Solutions, v.makeSolution(c, accountCtx, solution, false))
		}
	}
	if err := solutions.Err(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeSolution(c echo.Context) error {
	solution, ok := c.Get(solutionKey).(models.Solution)
	if !ok {
		c.Logger().Error("solution not extracted")
		return fmt.Errorf("solution not extracted")
	}
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		c.Logger().Error("auth not extracted")
		return fmt.Errorf("auth not extracted")
	}
	return c.JSON(http.StatusOK, v.makeSolution(c, accountCtx, solution, true))
}

func (v *View) extractSolution(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("solution"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Invalid solution ID."),
			}
		}
		if err := syncStore(c, v.core.Solutions); err != nil {
			return err
		}
		solution, err := v.core.Solutions.Get(getContext(c), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: localize(c, "Solution not found."),
				}
			}
			c.Logger().Error(err)
			return err
		}
		accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
		if !ok {
			c.Logger().Error("auth not extracted")
			return fmt.Errorf("auth not extracted")
		}
		c.Set(solutionKey, solution)
		c.Set(permissionCtxKey, v.getSolutionPermissions(accountCtx, solution))
		return next(c)
	}
}

func (v *View) getSolutionPermissions(
	ctx *managers.AccountContext, solution models.Solution,
) managers.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if solution.ID == 0 {
		return permissions
	}
	if account := ctx.Account; account != nil &&
		solution.AuthorID != 0 && account.ID == int64(solution.AuthorID) {
		permissions[models.ObserveSolutionRole] = struct{}{}
	}
	return permissions
}

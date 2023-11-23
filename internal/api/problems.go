package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
)

// registerProblemHandlers registers handlers for problem management.
func (v *View) registerProblemHandlers(g *echo.Group) {
	g.GET(
		"/v0/problems", v.observeProblems,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(perms.ObserveProblemsRole),
	)
	g.POST(
		"/v0/problems", v.createProblem,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(perms.CreateProblemRole),
	)
	g.GET(
		"/v0/problems/:problem", v.observeProblem,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractProblem,
		v.requirePermission(perms.ObserveProblemRole),
	)
	g.PATCH(
		"/v0/problems/:problem", v.updateProblem,
		v.extractAuth(v.sessionAuth), v.extractProblem,
		v.requirePermission(perms.UpdateProblemRole),
	)
	g.POST(
		"/v0/problems/:problem/rebuild", v.rebuildProblem,
		v.extractAuth(v.sessionAuth), v.extractProblem,
		v.requirePermission(perms.UpdateProblemRole),
	)
	g.GET(
		"/v0/problems/:problem/resources/:resource", v.observeProblemResource,
		v.extractAuth(v.sessionAuth), v.extractProblem,
		v.requirePermission(perms.ObserveProblemRole),
	)
	g.DELETE(
		"/v0/problems/:problem", v.deleteProblem,
		v.extractAuth(v.sessionAuth), v.extractProblem,
		v.requirePermission(perms.DeleteProblemRole),
	)
}

type ProblemStatement = models.ProblemStatementConfig

type ProblemTask struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type Problem struct {
	ID          int64                 `json:"id"`
	Title       string                `json:"title"`
	Statement   *ProblemStatement     `json:"statement,omitempty"`
	Config      *models.ProblemConfig `json:"config,omitempty"`
	Permissions []string              `json:"permissions,omitempty"`
	LastTask    *ProblemTask          `json:"last_task,omitempty"`
}

type Problems struct {
	Problems []Problem `json:"problems"`
}

var problemPermissions = []string{
	perms.UpdateProblemRole,
	perms.DeleteProblemRole,
}

func (v *View) makeProblem(
	c echo.Context,
	problem models.Problem,
	permissions perms.Permissions,
	withStatement bool,
	withTask bool,
	locales map[string]struct{},
) Problem {
	resp := Problem{
		ID:    problem.ID,
		Title: problem.Title,
	}
	if withStatement {
		config, err := problem.GetConfig()
		if err == nil {
			resp.Config = &models.ProblemConfig{
				TimeLimit:   config.TimeLimit,
				MemoryLimit: config.MemoryLimit,
			}
		}
	}
	locale := getLocale(c)
	if resources, err := v.core.ProblemResources.FindByProblem(
		getContext(c), problem.ID,
	); err == nil {
		for _, resource := range resources {
			if resource.Kind != models.ProblemStatement {
				continue
			}
			var config models.ProblemStatementConfig
			if err := resource.ScanConfig(&config); err != nil {
				continue
			}
			if len(locales) > 0 {
				if _, ok := locales[config.Locale]; !ok {
					continue
				}
			}
			if resp.Statement == nil || config.Locale == locale.Name() {
				statement := ProblemStatement{
					Locale: config.Locale,
					Title:  config.Title,
				}
				if withStatement {
					statement.Legend = config.Legend
					statement.Input = config.Input
					statement.Output = config.Output
					statement.Notes = config.Notes
					statement.Samples = config.Samples
					statement.Scoring = config.Scoring
					statement.Interaction = config.Interaction
				}
				resp.Statement = &statement
			}
			if config.Locale == locale.Name() {
				break
			}
		}
	}
	if withTask && permissions.HasPermission(perms.UpdateProblemRole) {
		task, err := v.findProblemTask(c, problem.ID)
		if err == nil {
			taskResp := ProblemTask{
				Status: task.Status.String(),
			}
			var state models.UpdateProblemPackageTaskState
			if err := task.ScanState(&state); err == nil {
				taskResp.Error = state.Error
			}
			resp.LastTask = &taskResp
		}
	}
	for _, permission := range problemPermissions {
		if permissions.HasPermission(permission) {
			resp.Permissions = append(resp.Permissions, permission)
		}
	}
	return resp
}

type problemFilter struct {
	Query string `query:"q"`
}

func (f *problemFilter) Filter(problem models.Problem) bool {
	if len(f.Query) > 0 {
		switch {
		case strings.HasPrefix(fmt.Sprint(problem.ID), f.Query):
		case strings.Contains(problem.Title, f.Query):
		default:
			return false
		}
	}
	return true
}

func (v *View) observeProblems(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	var filter problemFilter
	if err := c.Bind(&filter); err != nil {
		c.Logger().Warn(err)
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Invalid filter."),
		}
	}
	problems, err := v.core.Problems.ReverseAll(getContext(c), 0)
	if err != nil {
		return err
	}
	defer func() { _ = problems.Close() }()
	var resp Problems
	for problems.Next() {
		problem := problems.Row()
		if !filter.Filter(problem) {
			continue
		}
		permissions := v.getProblemPermissions(accountCtx, problem)
		if permissions.HasPermission(perms.ObserveProblemRole) {
			resp.Problems = append(
				resp.Problems,
				v.makeProblem(c, problem, permissions, false, false, nil),
			)
		}
	}
	if err := problems.Err(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeProblem(c echo.Context) error {
	problem, ok := c.Get(problemKey).(models.Problem)
	if !ok {
		return fmt.Errorf("problem not extracted")
	}
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	permissions := v.getProblemPermissions(accountCtx, problem)
	return c.JSON(
		http.StatusOK,
		v.makeProblem(c, problem, permissions, true, true, nil),
	)
}

func (v *View) observeProblemResource(c echo.Context) error {
	problem, ok := c.Get(problemKey).(models.Problem)
	if !ok {
		return fmt.Errorf("problem not extracted")
	}
	resourceName := c.Param("resource")
	locale := getLocale(c)
	resources, err := v.core.ProblemResources.FindByProblem(getContext(c), problem.ID)
	if err != nil {
		return err
	}
	var foundResource *models.ProblemResource
	for i, resource := range resources {
		if resource.Kind != models.ProblemStatementResource {
			continue
		}
		if resource.FileID == 0 {
			continue
		}
		config := models.ProblemStatementResourceConfig{}
		if err := resource.ScanConfig(&config); err != nil {
			continue
		}
		if config.Name != resourceName {
			continue
		}
		if foundResource == nil || config.Locale == locale.Name() {
			foundResource = &resources[i]
		}
	}
	if foundResource == nil {
		return errorResponse{
			Code:    http.StatusNotFound,
			Message: localize(c, "File not found."),
		}
	}
	file, err := v.core.Files.Get(getContext(c), int64(foundResource.FileID))
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResponse{
				Code:    http.StatusNotFound,
				Message: localize(c, "File not found."),
			}
		}
		return err
	}
	c.Set(fileKey, file)
	return v.observeFileContent(c)
}

type UpdateProblemForm struct {
	Title       *string     `json:"title" form:"title"`
	OwnerID     *int64      `json:"owner_id" form:"owner_id"`
	PackageFile *FileReader `json:"-"`
}

func (f *UpdateProblemForm) Close() error {
	if f.PackageFile == nil {
		return nil
	}
	return f.PackageFile.Close()
}

func (f *UpdateProblemForm) Parse(c echo.Context) error {
	if err := c.Bind(f); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	formFile, err := c.FormFile("file")
	if err != nil {
		if err != http.ErrMissingFile {
			return err
		}
	} else {
		file, err := managers.NewMultipartFileReader(formFile)
		if err != nil {
			return err
		}
		f.PackageFile = file
	}
	return nil
}

func (f *UpdateProblemForm) Update(c echo.Context, problem *models.Problem) error {
	errors := errorFields{}
	if f.Title != nil {
		if len(*f.Title) < 4 {
			errors["title"] = errorField{
				Message: localize(c, "Title is too short."),
			}
		} else if len(*f.Title) > 64 {
			errors["title"] = errorField{
				Message: localize(c, "Title is too long."),
			}
		}
		problem.Title = *f.Title
	}
	if len(errors) > 0 {
		return errorResponse{
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	return nil
}

type CreateProblemForm struct {
	UpdateProblemForm
}

func (f *CreateProblemForm) Update(c echo.Context, problem *models.Problem) error {
	if f.Title == nil {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Form has invalid fields."),
			InvalidFields: errorFields{
				"title": errorField{Message: localize(c, "Title is required.")},
			},
		}
	}
	if f.PackageFile == nil {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Form has invalid fields."),
			InvalidFields: errorFields{
				"file": errorField{Message: localize(c, "File is required.")},
			},
		}
	}
	return f.UpdateProblemForm.Update(c, problem)
}

func (v *View) createProblem(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	var form CreateProblemForm
	if err := form.Parse(c); err != nil {
		return err
	}
	defer func() { _ = form.Close() }()
	var problem models.Problem
	if err := form.Update(c, &problem); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	if account := accountCtx.Account; account != nil {
		problem.OwnerID = NInt64(account.ID)
	}
	file, err := v.files.UploadFile(getContext(c), form.PackageFile)
	if err != nil {
		return err
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		if err := v.files.ConfirmUploadFile(ctx, &file); err != nil {
			return err
		}
		problem.PackageID = models.NInt64(file.ID)
		if err := v.core.Problems.Create(ctx, &problem); err != nil {
			return err
		}
		task := models.Task{}
		if err := task.SetConfig(models.UpdateProblemPackageTaskConfig{
			ProblemID: problem.ID,
			FileID:    file.ID,
			Compile:   true,
		}); err != nil {
			return err
		}
		return v.core.Tasks.Create(ctx, &task)
	}, sqlRepeatableRead); err != nil {
		return err
	}
	permissions := v.getProblemPermissions(accountCtx, problem)
	return c.JSON(
		http.StatusCreated,
		v.makeProblem(c, problem, permissions, false, false, nil),
	)
}

func (v *View) updateProblem(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	problem, ok := c.Get(problemKey).(models.Problem)
	if !ok {
		return fmt.Errorf("problem not extracted")
	}
	permissions := v.getProblemPermissions(accountCtx, problem)
	var form UpdateProblemForm
	if err := form.Parse(c); err != nil {
		return err
	}
	defer func() { _ = form.Close() }()
	if err := form.Update(c, &problem); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	var missingPermissions []string
	if form.OwnerID != nil {
		if !permissions.HasPermission(perms.UpdateProblemOwnerRole) {
			missingPermissions = append(missingPermissions, perms.UpdateProblemOwnerRole)
		} else {
			account, err := v.core.Accounts.Get(getContext(c), *form.OwnerID)
			if err != nil {
				if err == sql.ErrNoRows {
					return errorResponse{
						Code:    http.StatusBadRequest,
						Message: localize(c, "User not found."),
					}
				}
				return err
			}
			if account.Kind != models.UserAccount {
				return errorResponse{
					Code:    http.StatusBadRequest,
					Message: localize(c, "User not found."),
				}
			}
			problem.OwnerID = models.NInt64(*form.OwnerID)
		}
	}
	if len(missingPermissions) > 0 {
		return errorResponse{
			Code:               http.StatusForbidden,
			Message:            localize(c, "Account missing permissions."),
			MissingPermissions: missingPermissions,
		}
	}
	var formFile *models.File
	if form.PackageFile != nil {
		file, err := v.files.UploadFile(getContext(c), form.PackageFile)
		if err != nil {
			return err
		}
		formFile = &file
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		if formFile != nil {
			if err := v.files.ConfirmUploadFile(ctx, formFile); err != nil {
				return err
			}
			problem.PackageID = models.NInt64(formFile.ID)
			task := models.Task{}
			if err := task.SetConfig(models.UpdateProblemPackageTaskConfig{
				ProblemID: problem.ID,
				FileID:    formFile.ID,
				Compile:   true,
			}); err != nil {
				return err
			}
			if err := v.core.Tasks.Create(ctx, &task); err != nil {
				return err
			}
		}
		return v.core.Problems.Update(ctx, problem)
	}, sqlRepeatableRead); err != nil {
		return err
	}
	return c.JSON(
		http.StatusOK,
		v.makeProblem(c, problem, permissions, false, false, nil),
	)
}

type RebuildProblemForm struct {
	Compile bool `json:"compile"`
}

func (v *View) rebuildProblem(c echo.Context) error {
	var form RebuildProblemForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	problem, ok := c.Get(problemKey).(models.Problem)
	if !ok {
		return fmt.Errorf("problem not extracted")
	}
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	permissions := v.getProblemPermissions(accountCtx, problem)
	if problem.PackageID == 0 {
		return c.JSON(
			http.StatusForbidden,
			v.makeProblem(c, problem, permissions, false, false, nil),
		)
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		task := models.Task{}
		if err := task.SetConfig(models.UpdateProblemPackageTaskConfig{
			ProblemID: problem.ID,
			FileID:    int64(problem.PackageID),
			Compile:   problem.CompiledID == 0 || form.Compile,
		}); err != nil {
			return err
		}
		return v.core.Tasks.Create(ctx, &task)
	}, sqlRepeatableRead); err != nil {
		return err
	}
	return c.JSON(
		http.StatusOK,
		v.makeProblem(c, problem, permissions, false, false, nil),
	)
}

func (v *View) deleteProblem(c echo.Context) error {
	problem, ok := c.Get(problemKey).(models.Problem)
	if !ok {
		return fmt.Errorf("problem not extracted")
	}
	solutions, err := v.core.Solutions.FindByProblem(problem.ID)
	if err != nil {
		return err
	}
	if len(solutions) > 0 {
		return errorResponse{
			Code: http.StatusForbidden,
		}
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		resources, err := v.core.ProblemResources.FindByProblem(ctx, problem.ID)
		if err != nil {
			return err
		}
		for _, resource := range resources {
			if err := v.core.ProblemResources.Delete(
				ctx, resource.ID,
			); err != nil {
				return err
			}
		}
		return v.core.Problems.Delete(ctx, problem.ID)
	}, sqlRepeatableRead); err != nil {
		return err
	}
	return c.JSON(
		http.StatusOK,
		v.makeProblem(c, problem, perms.PermissionSet{}, false, false, nil),
	)
}

func (v *View) findProblemTask(c echo.Context, id int64) (models.Task, error) {
	tasks, err := v.core.Tasks.FindByProblem(id)
	if err != nil {
		return models.Task{}, err
	}
	var lastTask models.Task
	for _, task := range tasks {
		if task.Kind == models.UpdateProblemPackageTask {
			var config models.UpdateProblemPackageTaskConfig
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

func (v *View) extractProblem(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("problem"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Invalid problem ID."),
			}
		}
		if err := syncStore(c, v.core.Problems); err != nil {
			return err
		}
		problem, err := v.core.Problems.Get(getContext(c), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: localize(c, "Problem not found."),
				}
			}
			return err
		}
		accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
		if !ok {
			return fmt.Errorf("account not extracted")
		}
		c.Set(problemKey, problem)
		c.Set(permissionCtxKey, v.getProblemPermissions(accountCtx, problem))
		return next(c)
	}
}

func (v *View) getProblemPermissions(
	ctx *managers.AccountContext, problem models.Problem,
) perms.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if account := ctx.Account; account != nil &&
		problem.OwnerID != 0 && account.ID == int64(problem.OwnerID) {
		permissions.AddPermission(
			perms.ObserveProblemRole,
			perms.UpdateProblemRole,
			perms.UpdateProblemOwnerRole,
			perms.DeleteProblemRole,
		)
	}
	return permissions
}

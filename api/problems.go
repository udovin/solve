package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

// registerProblemHandlers registers handlers for problem management.
func (v *View) registerProblemHandlers(g *echo.Group) {
	g.GET(
		"/v0/problems", v.observeProblems,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(models.ObserveProblemsRole),
	)
	g.POST(
		"/v0/problems", v.createProblem,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(models.CreateProblemRole),
	)
	g.GET(
		"/v0/problems/:problem", v.observeProblem,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractProblem,
		v.requirePermission(models.ObserveProblemRole),
	)
	g.PATCH(
		"/v0/problems/:problem", v.updateProblem,
		v.extractAuth(v.sessionAuth), v.extractProblem,
		v.requirePermission(models.UpdateProblemRole),
	)
	g.POST(
		"/v0/problems/:problem/rebuild", v.rebuildProblem,
		v.extractAuth(v.sessionAuth), v.extractProblem,
		v.requirePermission(models.UpdateProblemRole),
	)
	g.GET(
		"/v0/problems/:problem/resources/:resource", v.observeProblemResource,
		v.extractAuth(v.sessionAuth), v.extractProblem,
		v.requirePermission(models.ObserveProblemRole),
	)
	g.DELETE(
		"/v0/problems/:problem", v.deleteProblem,
		v.extractAuth(v.sessionAuth), v.extractProblem,
		v.requirePermission(models.DeleteProblemRole),
	)
}

type ProblemStatement = models.ProblemStatementConfig

type Problem struct {
	ID          int64                 `json:"id"`
	Title       string                `json:"title"`
	Statement   *ProblemStatement     `json:"statement,omitempty"`
	Config      *models.ProblemConfig `json:"config,omitempty"`
	Permissions []string              `json:"permissions,omitempty"`
}

type Problems struct {
	Problems []Problem `json:"problems"`
}

var problemPermissions = []string{
	models.UpdateProblemRole,
	models.DeleteProblemRole,
}

func (v *View) makeProblem(
	c echo.Context,
	problem models.Problem,
	permissions managers.Permissions,
	withStatement bool,
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
		problem.ID,
	); err == nil {
		for _, resource := range resources {
			if resource.Kind != models.ProblemStatement {
				continue
			}
			var config models.ProblemStatementConfig
			if err := resource.ScanConfig(&config); err != nil {
				continue
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
				}
				resp.Statement = &statement
			}
			if config.Locale == locale.Name() {
				break
			}
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
	problems, err := v.core.Problems.All()
	if err != nil {
		return err
	}
	var resp Problems
	for _, problem := range problems {
		if !filter.Filter(problem) {
			continue
		}
		permissions := v.getProblemPermissions(accountCtx, problem)
		if permissions.HasPermission(models.ObserveProblemRole) {
			resp.Problems = append(
				resp.Problems,
				v.makeProblem(c, problem, permissions, false),
			)
		}
	}
	sortFunc(resp.Problems, problemGreater)
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
		v.makeProblem(c, problem, permissions, true),
	)
}

func (v *View) observeProblemResource(c echo.Context) error {
	problem, ok := c.Get(problemKey).(models.Problem)
	if !ok {
		return fmt.Errorf("problem not extracted")
	}
	resourceName := c.Param("resource")
	locale := getLocale(c)
	resources, err := v.core.ProblemResources.FindByProblem(problem.ID)
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
	file, err := v.core.Files.Get(int64(foundResource.FileID))
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
		v.makeProblem(c, problem, permissions, false),
	)
}

func (v *View) updateProblem(c echo.Context) error {
	problem, ok := c.Get(problemKey).(models.Problem)
	if !ok {
		return fmt.Errorf("problem not extracted")
	}
	var form UpdateProblemForm
	if err := form.Parse(c); err != nil {
		return err
	}
	defer func() { _ = form.Close() }()
	if err := form.Update(c, &problem); err != nil {
		return c.JSON(http.StatusBadRequest, err)
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
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	permissions := v.getProblemPermissions(accountCtx, problem)
	return c.JSON(
		http.StatusOK,
		v.makeProblem(c, problem, permissions, false),
	)
}

func (v *View) rebuildProblem(c echo.Context) error {
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
			v.makeProblem(c, problem, permissions, false),
		)
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		task := models.Task{}
		if err := task.SetConfig(models.UpdateProblemPackageTaskConfig{
			ProblemID: problem.ID,
			FileID:    int64(problem.PackageID),
			Compile:   problem.CompiledID == 0,
		}); err != nil {
			return err
		}
		return v.core.Tasks.Create(ctx, &task)
	}, sqlRepeatableRead); err != nil {
		return err
	}
	return c.JSON(
		http.StatusOK,
		v.makeProblem(c, problem, permissions, false),
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
		resources, err := v.core.ProblemResources.FindByProblem(problem.ID)
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
		v.makeProblem(c, problem, managers.PermissionSet{}, false),
	)
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
		problem, err := v.core.Problems.Get(id)
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
) managers.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if account := ctx.Account; account != nil &&
		problem.OwnerID != 0 && account.ID == int64(problem.OwnerID) {
		permissions[models.ObserveProblemRole] = struct{}{}
		permissions[models.UpdateProblemRole] = struct{}{}
		permissions[models.DeleteProblemRole] = struct{}{}
	}
	return permissions
}

func problemGreater(l, r Problem) bool {
	return l.ID > r.ID
}

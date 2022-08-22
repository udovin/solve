package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

func (v *View) registerCompilerHandlers(g *echo.Group) {
	g.GET(
		"/v0/compilers", v.ObserveCompilers,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(models.ObserveCompilersRole),
	)
	g.POST(
		"/v0/compilers", v.createCompiler,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(models.CreateCompilerRole),
	)
	g.PATCH(
		"/v0/compilers/:compiler", v.updateCompiler,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(models.UpdateCompilerRole),
	)
	g.DELETE(
		"/v0/compilers/:compiler", v.deleteCompiler,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(models.DeleteCompilerRole),
	)
}

type Compiler struct {
	ID     int64       `json:"id"`
	Name   string      `json:"name"`
	Config models.JSON `json:"config"`
}

type Compilers struct {
	Compilers []Compiler `json:"compilers"`
}

type compilerSorter []Compiler

func (v compilerSorter) Len() int {
	return len(v)
}

func (v compilerSorter) Less(i, j int) bool {
	return v[i].ID > v[j].ID
}

func (v compilerSorter) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

// ObserveCompilers returns list of available compilers.
func (v *View) ObserveCompilers(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	compilers, err := v.core.Compilers.All()
	if err != nil {
		return err
	}
	var resp Compilers
	for _, compiler := range compilers {
		permissions := v.getCompilerPermissions(accountCtx, compiler)
		if permissions.HasPermission(models.ObserveCompilerRole) {
			resp.Compilers = append(resp.Compilers, makeCompiler(compiler))
		}
	}
	sort.Sort(compilerSorter(resp.Compilers))
	return c.JSON(http.StatusOK, resp)
}

type createCompilerForm struct {
	Name   string                `form:"name" json:"name"`
	Config models.CompilerConfig `form:"config" json:"config"`
}

func (f *createCompilerForm) Update(compiler *models.Compiler) error {
	errors := errorFields{}
	if len(f.Name) < 4 {
		errors["name"] = errorField{Message: "name is too short"}
	}
	if len(f.Name) > 64 {
		errors["name"] = errorField{Message: "name is too long"}
	}
	compiler.Name = f.Name
	if err := compiler.SetConfig(f.Config); err != nil {
		errors["config"] = errorField{Message: "invalid config"}
	}
	if len(errors) > 0 {
		return &errorResponse{
			Code:          http.StatusBadRequest,
			Message:       "form has invalid fields",
			InvalidFields: errors,
		}
	}
	return nil
}

func (v *View) createCompiler(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	var form createCompilerForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var compiler models.Compiler
	if err := form.Update(&compiler); err != nil {
		return err
	}
	if account := accountCtx.Account; account != nil {
		compiler.OwnerID = models.NInt64(account.ID)
	}
	formFile, err := c.FormFile("file")
	if err != nil {
		return err
	}
	file, err := v.Files.UploadFile(getContext(c), formFile)
	if err != nil {
		return err
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		if err := v.Files.ConfirmUploadFile(ctx, &file); err != nil {
			return err
		}
		compiler.ImageID = file.ID
		return v.core.Compilers.Create(ctx, &compiler)
	}, sqlRepeatableRead); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, makeCompiler(compiler))
}

func (v *View) updateCompiler(c echo.Context) error {
	return errNotImplemented
}

func (v *View) deleteCompiler(c echo.Context) error {
	compiler, ok := c.Get(compilerKey).(models.Compiler)
	if !ok {
		return fmt.Errorf("compiler not extracted")
	}
	if err := v.core.Compilers.Delete(getContext(c), compiler.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, makeCompiler(compiler))
}

func makeCompiler(compiler models.Compiler) Compiler {
	return Compiler{
		ID:     compiler.ID,
		Name:   compiler.Name,
		Config: compiler.Config,
	}
}

func (v *View) getCompilerPermissions(
	ctx *managers.AccountContext, compiler models.Compiler,
) managers.PermissionSet {
	permissions := ctx.Permissions.Clone()
	permissions[models.ObserveCompilerRole] = struct{}{}
	return permissions
}

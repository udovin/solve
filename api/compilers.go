package api

import (
	"context"
	"fmt"
	"net/http"

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
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Config JSON   `json:"config"`
}

type Compilers struct {
	Compilers []Compiler `json:"compilers"`
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
	sortFunc(resp.Compilers, compilerGreater)
	return c.JSON(http.StatusOK, resp)
}

type CreateCompilerForm struct {
	Name      string      `form:"name" json:"name"`
	Config    JSON        `form:"config" json:"config"`
	ImageFile *FileReader `json:"-"`
}

func (f *CreateCompilerForm) Parse(c echo.Context) error {
	if err := c.Bind(f); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	formFile, err := c.FormFile("file")
	if err != nil {
		return err
	}
	file, err := managers.NewMultipartFileReader(formFile)
	if err != nil {
		return err
	}
	f.ImageFile = file
	return nil
}

func (f *CreateCompilerForm) Update(compiler *models.Compiler) error {
	errors := errorFields{}
	if len(f.Name) < 4 {
		errors["name"] = errorField{Message: "name is too short"}
	}
	if len(f.Name) > 64 {
		errors["name"] = errorField{Message: "name is too long"}
	}
	compiler.Name = f.Name
	compiler.Config = f.Config.JSON
	if config, err := compiler.GetConfig(); err != nil {
		errors["name"] = errorField{Message: "invalid config"}
	} else if err := compiler.SetConfig(config); err != nil {
		errors["name"] = errorField{Message: "invalid config"}
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
	var form CreateCompilerForm
	if err := form.Parse(c); err != nil {
		return err
	}
	defer func() { _ = form.ImageFile.Close() }()
	var compiler models.Compiler
	if err := form.Update(&compiler); err != nil {
		return err
	}
	if account := accountCtx.Account; account != nil {
		compiler.OwnerID = models.NInt64(account.ID)
	}
	file, err := v.files.UploadFile(getContext(c), form.ImageFile)
	if err != nil {
		return err
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		if err := v.files.ConfirmUploadFile(ctx, &file); err != nil {
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
		Config: JSON{compiler.Config},
	}
}

func (v *View) getCompilerPermissions(
	ctx *managers.AccountContext, compiler models.Compiler,
) managers.PermissionSet {
	permissions := ctx.Permissions.Clone()
	permissions[models.ObserveCompilerRole] = struct{}{}
	return permissions
}

func compilerGreater(l, r Compiler) bool {
	return l.ID > r.ID
}

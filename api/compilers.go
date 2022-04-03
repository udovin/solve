package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

func (v *View) registerCompilerHandlers(g *echo.Group) {
	g.GET(
		"/v0/compilers", v.ObserveCompilers,
		v.sessionAuth,
		v.requireAuthRole(models.ObserveCompilersRole),
	)
	g.POST(
		"/v0/compiler", v.createCompiler,
		v.sessionAuth, v.requireAuth,
		v.requireAuthRole(models.CreateCompilerRole),
	)
	g.PATCH(
		"/v0/compiler/:compiler", v.updateCompiler,
		v.sessionAuth, v.requireAuth,
		v.requireAuthRole(models.UpdateCompilerRole),
	)
	g.DELETE(
		"/v0/compiler/:compiler", v.deleteCompiler,
		v.sessionAuth, v.requireAuth,
		v.requireAuthRole(models.DeleteCompilerRole),
	)
}

type Compiler struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
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
	roles, ok := c.Get(authRolesKey).(core.RoleSet)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	var resp Compilers
	compilers, err := v.core.Compilers.All()
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	for _, compiler := range compilers {
		compilerRoles := v.extendCompilerRoles(c, roles, compiler)
		if ok, err := v.core.HasRole(compilerRoles, models.ObserveCompilerRole); ok && err == nil {
			resp.Compilers = append(resp.Compilers, makeCompiler(compiler))
		}
	}
	sort.Sort(compilerSorter(resp.Compilers))
	return c.JSON(http.StatusOK, resp)
}

type createCompilerForm struct {
	Name string `form:"name"`
}

func (f createCompilerForm) validate() *errorResponse {
	errors := errorFields{}
	if len(f.Name) < 4 {
		errors["name"] = errorField{Message: "name is too short"}
	}
	if len(f.Name) > 64 {
		errors["name"] = errorField{Message: "name is too long"}
	}
	if len(errors) > 0 {
		return &errorResponse{
			Message:       "form has invalid fields",
			InvalidFields: errors,
		}
	}
	return nil
}

func (f createCompilerForm) Update(compiler *models.Compiler) *errorResponse {
	if err := f.validate(); err != nil {
		return err
	}
	compiler.Name = f.Name
	return nil
}

func (v *View) createCompiler(c echo.Context) error {
	var form createCompilerForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var compiler models.Compiler
	if err := form.Update(&compiler); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	if account, ok := c.Get(authAccountKey).(models.Account); ok {
		compiler.OwnerID = models.NInt64(account.ID)
	}
	if err := v.core.WrapTx(c.Request().Context(), func(ctx context.Context) error {
		file, err := c.FormFile("file")
		if err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer func() {
			_ = src.Close()
		}()
		if err := v.core.Compilers.Create(ctx, &compiler); err != nil {
			return err
		}
		dst, err := os.Create(filepath.Join(
			v.core.Config.Storage.CompilersDir,
			fmt.Sprintf("%d.zip", compiler.ID),
		))
		if err != nil {
			return err
		}
		defer dst.Close()
		_, err = io.Copy(dst, src)
		return err
	}, sqlRepeatableRead); err != nil {
		c.Logger().Error(err)
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
		c.Logger().Error("problem not extracted")
		return fmt.Errorf("problem not extracted")
	}
	if err := v.core.Compilers.Delete(c.Request().Context(), compiler.ID); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, makeCompiler(compiler))
}

func makeCompiler(compiler models.Compiler) Compiler {
	return Compiler{
		ID:   compiler.ID,
		Name: compiler.Name,
	}
}

func (v *View) extendCompilerRoles(
	c echo.Context, roles core.RoleSet, compiler models.Compiler,
) core.RoleSet {
	compilerRoles := roles.Clone()
	if compiler.ID == 0 {
		return compilerRoles
	}
	addRole := func(name string) {
		if err := v.core.AddRole(compilerRoles, name); err != nil {
			c.Logger().Error(err)
		}
	}
	addRole(models.ObserveCompilerRole)
	return compilerRoles
}

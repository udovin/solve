package api

import (
	"fmt"
	"net/http"
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
			resp.Compilers = append(resp.Compilers, makeCompiler(compiler, compilerRoles, v.core))
		}
	}
	sort.Sort(compilerSorter(resp.Compilers))
	return c.JSON(http.StatusOK, resp)
}

func (v *View) createCompiler(c echo.Context) error {
	return errNotImplemented
}

func (v *View) updateCompiler(c echo.Context) error {
	return errNotImplemented
}

func (v *View) deleteCompiler(c echo.Context) error {
	return errNotImplemented
}

func makeCompiler(compiler models.Compiler, roles core.RoleSet, core *core.Core) Compiler {
	return Compiler{
		ID:   compiler.ID,
		Name: "todo", //compiler.Name,
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

package api

import (
	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/models"
)

func (v *View) registerCompilerHandlers(g *echo.Group) {
	g.GET(
		"/v0/compilers", v.observeCompilers,
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

func (v *View) observeCompilers(c echo.Context) error {
	return errNotImplemented
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

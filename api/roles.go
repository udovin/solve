package api

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/udovin/solve/models"
)

// registerUserHandlers registers handlers for user management.
func (v *View) registerRoleHandlers(g *echo.Group) {
	g.GET(
		"/roles", v.observeRoles,
		v.requireAuth(v.sessionAuth),
		v.requireRole(models.ObserveRoleRole),
	)
	g.POST(
		"/roles", v.createRole,
		v.requireAuth(v.sessionAuth),
		v.requireRole(models.CreateRoleRole),
	)
	g.DELETE(
		"/roles/:role", v.deleteRole,
		v.requireAuth(v.userAuth),
		v.requireRole(models.DeleteRoleRole),
	)
	g.GET(
		"/users/:user/roles", v.observeUserRoles,
		v.requireAuth(v.sessionAuth),
		v.requireRole(models.ObserveUserRoleRole),
	)
	g.POST(
		"/users/:user/roles", v.createUserRole,
		v.requireAuth(v.sessionAuth),
		v.requireRole(models.CreateUserRoleRole),
	)
	g.DELETE(
		"/users/:user/roles/:role", v.deleteUserRole,
		v.requireAuth(v.sessionAuth),
		v.requireRole(models.DeleteUserRoleRole),
	)
}

var errNotImplemented = fmt.Errorf("not implemented")

func (v *View) createRole(c echo.Context) error {
	return errNotImplemented
}

func (v *View) deleteRole(c echo.Context) error {
	return errNotImplemented
}

func (v *View) observeRoles(c echo.Context) error {
	return errNotImplemented
}

func (v *View) createUserRole(c echo.Context) error {
	return errNotImplemented
}

func (v *View) deleteUserRole(c echo.Context) error {
	return errNotImplemented
}

func (v *View) observeUserRoles(c echo.Context) error {
	return errNotImplemented
}

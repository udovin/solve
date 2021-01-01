package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

// Role represents role.
type Role struct {
	// ID contains role ID.
	ID int64 `json:"role"`
	// Code contains role code.
	Code string `json:"code"`
}

// registerUserHandlers registers handlers for user management.
func (v *View) registerRoleHandlers(g *echo.Group) {
	g.GET(
		"/roles", v.observeRoles,
		v.sessionAuth, v.requireAuth,
		v.requireAuthRole(models.ObserveRoleRole),
	)
	g.POST(
		"/roles", v.createRole,
		v.sessionAuth, v.requireAuth,
		v.requireAuthRole(models.CreateRoleRole),
	)
	g.DELETE(
		"/roles/:role", v.deleteRole,
		v.sessionAuth, v.requireAuth, v.extractRole,
		v.requireAuthRole(models.DeleteRoleRole),
	)
	g.GET(
		"/roles/:role/roles", v.observeRoleRoles,
		v.sessionAuth, v.requireAuth, v.extractRole,
		v.requireAuthRole(models.ObserveRoleRole),
	)
	g.GET(
		"/users/:user/roles", v.observeUserRoles,
		v.sessionAuth, v.requireAuth, v.extractUser,
		v.requireAuthRole(models.ObserveUserRoleRole),
	)
	g.POST(
		"/users/:user/roles", v.createUserRole,
		v.sessionAuth, v.requireAuth, v.extractUser,
		v.requireAuthRole(models.CreateUserRoleRole),
	)
	g.DELETE(
		"/users/:user/roles/:role", v.deleteUserRole,
		v.sessionAuth, v.requireAuth, v.extractUser, v.extractRole,
		v.requireAuthRole(models.DeleteUserRoleRole),
	)
}

var errNotImplemented = fmt.Errorf("not implemented")

func (v *View) observeRoles(c echo.Context) error {
	var resp []Role
	roles, err := v.core.Roles.All()
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	for _, role := range roles {
		resp = append(resp, Role{
			ID:   role.ID,
			Code: role.Code,
		})
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) createRole(c echo.Context) error {
	return errNotImplemented
}

func (v *View) deleteRole(c echo.Context) error {
	return errNotImplemented
}

func (v *View) observeRoleRoles(c echo.Context) error {
	return errNotImplemented
}

func (v *View) observeUserRoles(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		c.Logger().Error("user not extracted")
		return fmt.Errorf("user not extracted")
	}
	edges, err := v.core.AccountRoles.FindByAccount(user.AccountID)
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	var resp []Role
	for _, edge := range edges {
		role, err := v.core.Roles.Get(edge.RoleID)
		if err != nil {
			if err == sql.ErrNoRows {
				c.Logger().Warnf("Role %v not found", edge.RoleID)
				continue
			}
			c.Logger().Error(err)
			return err
		}
		resp = append(resp, Role{
			ID:   role.ID,
			Code: role.Code,
		})
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) createUserRole(c echo.Context) error {
	return errNotImplemented
}

func (v *View) deleteUserRole(c echo.Context) error {
	return errNotImplemented
}

func (v *View) extractRole(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("role"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return err
		}
		role, err := v.core.Roles.Get(id)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.NoContent(http.StatusNotFound)
			}
			c.Logger().Error(err)
			return err
		}
		c.Set(roleKey, role)
		return next(c)
	}
}

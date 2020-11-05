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
		v.extractRole,
	)
	g.GET(
		"/roles/:role/roles", v.observeRoleRoles,
		v.requireAuth(v.userAuth),
		v.requireRole(models.ObserveRoleRole),
		v.extractRole,
	)
	g.GET(
		"/users/:user/roles", v.observeUserRoles,
		v.requireAuth(v.sessionAuth),
		v.requireRole(models.ObserveUserRoleRole),
		v.extractUser,
	)
	g.POST(
		"/users/:user/roles", v.createUserRole,
		v.requireAuth(v.sessionAuth),
		v.requireRole(models.CreateUserRoleRole),
		v.extractUser,
	)
	g.DELETE(
		"/users/:user/roles/:role", v.deleteUserRole,
		v.requireAuth(v.sessionAuth),
		v.requireRole(models.DeleteUserRoleRole),
		v.extractUser,
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
	var resp []Role
	roles, err := v.core.Roles.All()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	for _, role := range roles {
		resp = append(resp, Role{
			ID:   role.ID,
			Code: role.Code,
		})
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeRoleRoles(c echo.Context) error {
	return errNotImplemented
}

func (v *View) createUserRole(c echo.Context) error {
	return errNotImplemented
}

func (v *View) deleteUserRole(c echo.Context) error {
	return errNotImplemented
}

func (v *View) observeUserRoles(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusInternalServerError)
	}
	edges, err := v.core.AccountRoles.FindByAccount(user.AccountID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
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
			return c.NoContent(http.StatusInternalServerError)
		}
		resp = append(resp, Role{
			ID:   role.ID,
			Code: role.Code,
		})
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) extractRole(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("role"), 10, 64)
		if err != nil {
			return err
		}
		role, err := v.core.Roles.Get(id)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.NoContent(http.StatusNotFound)
			}
		}
		c.Set(roleKey, role)
		return next(c)
	}
}

func (v *View) extractUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("user"), 10, 64)
		if err != nil {
			return err
		}
		user, err := v.core.Users.Get(id)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.NoContent(http.StatusNotFound)
			}
		}
		c.Set(userKey, user)
		return next(c)
	}
}

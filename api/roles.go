package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/models"
)

// Role represents role.
type Role struct {
	// ID contains role ID.
	ID int64 `json:"id"`
	// Code contains role code.
	Code string `json:"code"`
}

// registerUserHandlers registers handlers for user management.
func (v *View) registerRoleHandlers(g *echo.Group) {
	g.GET(
		"/roles", v.observeRoles,
		v.sessionAuth, v.requireAuth,
		v.requireAuthRole(models.ObserveRolesRole),
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
		v.requireAuthRole(models.ObserveRoleRolesRole),
	)
	g.GET(
		"/users/:user/roles", v.observeUserRoles,
		v.sessionAuth, v.requireAuth, v.extractUser,
		v.requireAuthRole(models.ObserveUserRolesRole),
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

// registerUserHandlers registers handlers for user management.
func (v *View) registerSocketRoleHandlers(g *echo.Group) {
	g.GET("/roles", v.observeRoles)
	g.POST("/roles", v.createRole)
	g.DELETE(
		"/roles/:role", v.deleteRole,
		v.extractRole,
	)
	g.GET(
		"/roles/:role/roles", v.observeRoleRoles,
		v.extractRole,
	)
	g.GET(
		"/users/:user/roles", v.observeUserRoles,
		v.extractUser,
	)
	g.POST(
		"/users/:user/roles", v.createUserRole,
		v.extractUser,
	)
	g.DELETE(
		"/users/:user/roles/:role", v.deleteUserRole,
		v.extractUser, v.extractRole,
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

type createRoleForm struct {
	Code string `json:"code"`
}

var roleCodeRegexp = regexp.MustCompile(
	`^[a-zA-Z]([a-zA-Z0-9_\\-])*[a-zA-Z0-9]$`,
)

func (f createRoleForm) validate() *errorResp {
	errors := errorFields{}
	if len(f.Code) < 3 {
		errors["code"] = errorField{Message: "code too short (<3)"}
	} else if len(f.Code) > 32 {
		errors["code"] = errorField{Message: "code too long (>32)"}
	} else if !roleCodeRegexp.MatchString(f.Code) {
		errors["code"] = errorField{Message: "code has invalid format"}
	}
	if len(errors) == 0 {
		return nil
	}
	return &errorResp{
		Message:       "passed invalid fields to form",
		InvalidFields: errors,
	}
}

func (f createRoleForm) Update(
	role *models.Role, roles *models.RoleStore,
) *errorResp {
	if err := f.validate(); err != nil {
		return err
	}
	role.Code = f.Code
	if _, err := roles.GetByCode(role.Code); err != sql.ErrNoRows {
		if err != nil {
			return &errorResp{Message: "unknown error"}
		}
		return &errorResp{
			Message: fmt.Sprintf("role %q already exists", role.Code),
		}
	}
	return nil
}

func (v *View) createRole(c echo.Context) error {
	var form createRoleForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var role models.Role
	if resp := form.Update(&role, v.core.Roles); resp != nil {
		return c.JSON(http.StatusBadRequest, resp)
	}
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		var err error
		role, err = v.core.Roles.CreateTx(tx, role)
		return err
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusCreated, Role{
		ID:   role.ID,
		Code: role.Code,
	})
}

func (v *View) deleteRole(c echo.Context) error {
	role, ok := c.Get(roleKey).(models.Role)
	if !ok {
		c.Logger().Error("role not extracted")
		return fmt.Errorf("role not extracted")
	}
	if role.IsBuiltIn() {
		return c.JSON(http.StatusBadRequest, errorResp{
			Message: "unable to delete builtin role",
		})
	}
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		return v.core.Roles.DeleteTx(tx, role.ID)
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, Role{
		ID:   role.ID,
		Code: role.Code,
	})
}

func (v *View) observeRoleRoles(c echo.Context) error {
	role, ok := c.Get(roleKey).(models.Role)
	if !ok {
		c.Logger().Error("role not extracted")
		return fmt.Errorf("role not extracted")
	}
	edges, err := v.core.RoleEdges.FindByRole(role.ID)
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	var resp []Role
	for _, edge := range edges {
		role, err := v.core.Roles.Get(edge.ChildID)
		if err != nil {
			if err == sql.ErrNoRows {
				c.Logger().Warnf("Role %v not found", edge.ChildID)
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

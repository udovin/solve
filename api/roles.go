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

// Roles represents roles response.
type Roles struct {
	Roles []Role `json:"roles"`
}

// registerUserHandlers registers handlers for user management.
func (v *View) registerRoleHandlers(g *echo.Group) {
	g.GET(
		"/v0/roles", v.observeRoles,
		v.sessionAuth, v.requireAuth,
		v.requireAuthRole(models.ObserveRolesRole),
	)
	g.POST(
		"/v0/roles", v.createRole,
		v.sessionAuth, v.requireAuth,
		v.requireAuthRole(models.CreateRoleRole),
	)
	g.DELETE(
		"/v0/roles/:role", v.deleteRole,
		v.sessionAuth, v.requireAuth, v.extractRole,
		v.requireAuthRole(models.DeleteRoleRole),
	)
	g.GET(
		"/v0/roles/:role/roles", v.observeRoleRoles,
		v.sessionAuth, v.requireAuth, v.extractRole,
		v.requireAuthRole(models.ObserveRoleRolesRole),
	)
	g.POST(
		"/v0/roles/:role/roles/:child_role", v.createRoleRole,
		v.sessionAuth, v.requireAuth, v.extractRole, v.extractChildRole,
		v.requireAuthRole(models.CreateRoleRoleRole),
	)
	g.DELETE(
		"/v0/roles/:role/roles/:child_role", v.deleteRoleRole,
		v.sessionAuth, v.requireAuth, v.extractRole, v.extractChildRole,
		v.requireAuthRole(models.DeleteRoleRoleRole),
	)
	g.GET(
		"/v0/users/:user/roles", v.observeUserRoles,
		v.sessionAuth, v.requireAuth, v.extractUser,
		v.requireAuthRole(models.ObserveUserRolesRole),
	)
	g.POST(
		"/v0/users/:user/roles/:role", v.createUserRole,
		v.sessionAuth, v.requireAuth, v.extractUser, v.extractRole,
		v.requireAuthRole(models.CreateUserRoleRole),
	)
	g.DELETE(
		"/v0/users/:user/roles/:role", v.deleteUserRole,
		v.sessionAuth, v.requireAuth, v.extractUser, v.extractRole,
		v.requireAuthRole(models.DeleteUserRoleRole),
	)
}

// registerUserHandlers registers handlers for user management.
func (v *View) registerSocketRoleHandlers(g *echo.Group) {
	g.GET("/v0/roles", v.observeRoles)
	g.POST("/v0/roles", v.createRole)
	g.DELETE(
		"/v0/roles/:role", v.deleteRole,
		v.extractRole,
	)
	g.GET(
		"/v0/roles/:role/roles", v.observeRoleRoles,
		v.extractRole,
	)
	g.POST(
		"/v0/roles/:role/roles/:child_role", v.createRoleRole,
		v.extractRole, v.extractChildRole,
	)
	g.DELETE(
		"/v0/roles/:role/roles/:child_role", v.deleteRoleRole,
		v.extractRole, v.extractChildRole,
	)
	g.GET(
		"/v0/users/:user/roles", v.observeUserRoles,
		v.extractUser,
	)
	g.POST(
		"/v0/users/:user/roles/:role", v.createUserRole,
		v.extractUser, v.extractRole,
	)
	g.DELETE(
		"/v0/users/:user/roles/:role", v.deleteUserRole,
		v.extractUser, v.extractRole,
	)
}

var errNotImplemented = fmt.Errorf("not implemented")

func (v *View) observeRoles(c echo.Context) error {
	var resp Roles
	roles, err := v.core.Roles.All()
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	for _, role := range roles {
		resp.Roles = append(resp.Roles, Role{
			ID:   role.ID,
			Code: role.Code,
		})
	}
	return c.JSON(http.StatusOK, resp)
}

type createRoleForm struct {
	Code string `json:"code" form:"code"`
}

var roleCodeRegexp = regexp.MustCompile(
	`^[a-zA-Z]([a-zA-Z0-9_\\-])*[a-zA-Z0-9]$`,
)

func (f createRoleForm) validate() *errorResponse {
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
	return &errorResponse{
		Message:       "passed invalid fields to form",
		InvalidFields: errors,
	}
}

func (f createRoleForm) Update(
	role *models.Role, roles *models.RoleStore,
) *errorResponse {
	if err := f.validate(); err != nil {
		return err
	}
	role.Code = f.Code
	if _, err := roles.GetByCode(role.Code); err != sql.ErrNoRows {
		if err != nil {
			return &errorResponse{Message: "unknown error"}
		}
		return &errorResponse{
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
		return c.JSON(http.StatusBadRequest, errorResponse{
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
	var resp Roles
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
		resp.Roles = append(resp.Roles, Role{
			ID:   role.ID,
			Code: role.Code,
		})
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) createRoleRole(c echo.Context) error {
	role, ok := c.Get(roleKey).(models.Role)
	if !ok {
		c.Logger().Error("role not extracted")
		return fmt.Errorf("role not extracted")
	}
	childRole, ok := c.Get(childRoleKey).(models.Role)
	if !ok {
		c.Logger().Error("child role not extracted")
		return fmt.Errorf("child role not extracted")
	}
	edges, err := v.core.RoleEdges.FindByRole(role.ID)
	if err != nil {
		return err
	}
	var resp Roles
	for _, edge := range edges {
		if edge.ChildID == childRole.ID {
			return c.JSON(http.StatusBadRequest, &errorResponse{
				Message: fmt.Sprintf(
					"role %q already has child %q",
					role.Code, childRole.Code,
				),
			})
		}
		role, err := v.core.Roles.Get(edge.ChildID)
		if err != nil {
			c.Logger().Error(err)
		} else {
			resp.Roles = append(resp.Roles, Role{
				ID:   role.ID,
				Code: role.Code,
			})
		}
	}
	edge := models.RoleEdge{
		RoleID:  role.ID,
		ChildID: childRole.ID,
	}
	if err := v.core.WithTx(c.Request().Context(),
		func(tx *sql.Tx) (err error) {
			edge, err = v.core.RoleEdges.CreateTx(tx, edge)
			return err
		},
	); err != nil {
		c.Logger().Error(err)
		return err
	}
	resp.Roles = append(resp.Roles, Role{
		ID:   childRole.ID,
		Code: childRole.Code,
	})
	return c.JSON(http.StatusCreated, resp)
}

func (v *View) deleteRoleRole(c echo.Context) error {
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
	var resp Roles
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
		resp.Roles = append(resp.Roles, Role{
			ID:   role.ID,
			Code: role.Code,
		})
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) createUserRole(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		c.Logger().Error("user not extracted")
		return fmt.Errorf("user not extracted")
	}
	role, ok := c.Get(roleKey).(models.Role)
	if !ok {
		c.Logger().Error("role not extracted")
		return fmt.Errorf("role not extracted")
	}
	edges, err := v.core.AccountRoles.FindByAccount(user.AccountID)
	if err != nil {
		return err
	}
	var resp Roles
	for _, edge := range edges {
		if edge.RoleID == role.ID {
			return c.JSON(http.StatusBadRequest, &errorResponse{
				Message: fmt.Sprintf(
					"user %q already has role %q",
					user.Login, role.Code,
				),
			})
		}
		role, err := v.core.Roles.Get(edge.RoleID)
		if err != nil {
			c.Logger().Error(err)
		} else {
			resp.Roles = append(resp.Roles, Role{
				ID:   role.ID,
				Code: role.Code,
			})
		}
	}
	edge := models.AccountRole{
		AccountID: user.AccountID,
		RoleID:    role.ID,
	}
	if err := v.core.WithTx(c.Request().Context(),
		func(tx *sql.Tx) (err error) {
			return v.core.AccountRoles.CreateTx(tx, &edge)
		},
	); err != nil {
		c.Logger().Error(err)
		return err
	}
	resp.Roles = append(resp.Roles, Role{
		ID:   role.ID,
		Code: role.Code,
	})
	return c.JSON(http.StatusCreated, resp)
}

func (v *View) deleteUserRole(c echo.Context) error {
	return errNotImplemented
}

func (v *View) extractRole(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := c.Param("role")
		id, err := strconv.ParseInt(code, 10, 64)
		if err != nil {
			role, err := v.core.Roles.GetByCode(code)
			if err != nil {
				if err == sql.ErrNoRows {
					resp := errorResponse{
						Message: fmt.Sprintf("role %q not found", code),
					}
					return c.JSON(http.StatusNotFound, resp)
				}
				c.Logger().Error(err)
				return err
			}
			c.Set(roleKey, role)
			return next(c)
		}
		role, err := v.core.Roles.Get(id)
		if err != nil {
			if err == sql.ErrNoRows {
				resp := errorResponse{
					Message: fmt.Sprintf("role %d not found", id),
				}
				return c.JSON(http.StatusNotFound, resp)
			}
			c.Logger().Error(err)
			return err
		}
		c.Set(roleKey, role)
		return next(c)
	}
}

func (v *View) extractChildRole(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := c.Param("child_role")
		id, err := strconv.ParseInt(code, 10, 64)
		if err != nil {
			role, err := v.core.Roles.GetByCode(code)
			if err != nil {
				if err == sql.ErrNoRows {
					resp := errorResponse{
						Message: fmt.Sprintf("role %q not found", code),
					}
					return c.JSON(http.StatusNotFound, resp)
				}
				c.Logger().Error(err)
				return err
			}
			c.Set(childRoleKey, role)
			return next(c)
		}
		role, err := v.core.Roles.Get(id)
		if err != nil {
			if err == sql.ErrNoRows {
				resp := errorResponse{
					Message: fmt.Sprintf("role %d not found", id),
				}
				return c.JSON(http.StatusNotFound, resp)
			}
			c.Logger().Error(err)
			return err
		}
		c.Set(childRoleKey, role)
		return next(c)
	}
}

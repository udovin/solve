package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/internal/core"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
)

// Role represents role.
type Role struct {
	// ID contains role ID.
	ID int64 `json:"id"`
	// Name contains role name.
	Name string `json:"name"`
	//
	BuiltIn bool `json:"built_in,omitempty"`
}

// Roles represents roles response.
type Roles struct {
	Roles []Role `json:"roles"`
}

// registerUserHandlers registers handlers for user management.
func (v *View) registerRoleHandlers(g *echo.Group) {
	g.GET(
		"/v0/roles", v.observeRoles,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(perms.ObserveRolesRole),
	)
	g.POST(
		"/v0/roles", v.createRole,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(perms.CreateRoleRole),
	)
	g.DELETE(
		"/v0/roles/:role", v.deleteRole,
		v.extractAuth(v.sessionAuth), v.extractRole,
		v.requirePermission(perms.DeleteRoleRole),
	)
	g.GET(
		"/v0/roles/:role/roles", v.observeRoleRoles,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractRole,
		v.requirePermission(perms.ObserveRoleRolesRole),
	)
	g.POST(
		"/v0/roles/:role/roles/:child_role", v.createRoleRole,
		v.extractAuth(v.sessionAuth), v.extractRole, v.extractChildRole,
		v.requirePermission(perms.CreateRoleRoleRole),
	)
	g.DELETE(
		"/v0/roles/:role/roles/:child_role", v.deleteRoleRole,
		v.extractAuth(v.sessionAuth), v.extractRole, v.extractChildRole,
		v.requirePermission(perms.DeleteRoleRoleRole),
	)
	g.GET(
		"/v0/users/:user/roles", v.observeUserRoles,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractUser,
		v.requirePermission(perms.ObserveUserRolesRole),
	)
	g.POST(
		"/v0/users/:user/roles/:role", v.createUserRole,
		v.extractAuth(v.sessionAuth), v.extractUser, v.extractRole,
		v.requirePermission(perms.CreateUserRoleRole),
	)
	g.DELETE(
		"/v0/users/:user/roles/:role", v.deleteUserRole,
		v.extractAuth(v.sessionAuth), v.extractUser, v.extractRole,
		v.requirePermission(perms.DeleteUserRoleRole),
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

func (v *View) observeRoles(c echo.Context) error {
	var resp Roles
	roles, err := v.core.Roles.ReverseAll(getContext(c), 0)
	if err != nil {
		return err
	}
	defer func() { _ = roles.Close() }()
	for roles.Next() {
		role := roles.Row()
		resp.Roles = append(resp.Roles, Role{
			ID:      role.ID,
			Name:    role.Name,
			BuiltIn: perms.IsBuiltInRole(role.Name),
		})
	}
	return c.JSON(http.StatusOK, resp)
}

type createRoleForm struct {
	Name string `json:"name" form:"name"`
}

var roleNameRegexp = regexp.MustCompile(
	`^[a-zA-Z]([a-zA-Z0-9_\\-])*[a-zA-Z0-9]$`,
)

func (f createRoleForm) Update(
	c echo.Context, role *models.Role, roles *models.RoleStore,
) error {
	errors := errorFields{}
	if len(f.Name) < 3 {
		errors["name"] = errorField{
			Message: localize(c, "Name is too short."),
		}
	} else if len(f.Name) > 64 {
		errors["name"] = errorField{
			Message: localize(c, "Name is too long."),
		}
	} else if !roleNameRegexp.MatchString(f.Name) {
		errors["name"] = errorField{
			Message: localize(c, "Name has invalid format."),
		}
	}
	if len(errors) > 0 {
		return errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	role.Name = f.Name
	ctx := getContext(c)
	if _, err := roles.GetByName(ctx, role.Name); err != sql.ErrNoRows {
		if err != nil {
			return err
		}
		return errorResponse{
			Code: http.StatusBadRequest,
			Message: localize(
				c, "Role \"{role}\" already exists.",
				replaceField("role", role.Name),
			),
		}
	}
	return nil
}

func (v *View) createRole(c echo.Context) error {
	var form createRoleForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Invalid form."),
		}
	}
	var role models.Role
	if err := form.Update(c, &role, v.core.Roles); err != nil {
		return err
	}
	if err := v.core.Roles.Create(getContext(c), &role); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, Role{
		ID:   role.ID,
		Name: role.Name,
	})
}

func (v *View) deleteRole(c echo.Context) error {
	role, ok := c.Get(roleKey).(models.Role)
	if !ok {
		return fmt.Errorf("role not extracted")
	}
	if perms.IsBuiltInRole(role.Name) {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Unable to delete builtin role."),
		}
	}
	if err := v.core.Roles.Delete(getContext(c), role.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, Role{
		ID:   role.ID,
		Name: role.Name,
	})
}

func (v *View) observeRoleRoles(c echo.Context) error {
	role, ok := c.Get(roleKey).(models.Role)
	if !ok {
		return fmt.Errorf("role not extracted")
	}
	if err := syncStore(c, v.core.RoleEdges); err != nil {
		return err
	}
	ctx := getContext(c)
	edges, err := v.core.RoleEdges.FindByRole(ctx, role.ID)
	if err != nil {
		return err
	}
	defer func() { _ = edges.Close() }()
	var resp Roles
	for edges.Next() {
		edge := edges.Row()
		role, err := v.core.Roles.Get(ctx, edge.ChildID)
		if err != nil {
			if err == sql.ErrNoRows {
				c.Logger().Warnf("Role %v not found", edge.ChildID)
				continue
			}
			return err
		}
		resp.Roles = append(resp.Roles, Role{
			ID:   role.ID,
			Name: role.Name,
		})
	}
	if err := edges.Err(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) createRoleRole(c echo.Context) error {
	role, ok := c.Get(roleKey).(models.Role)
	if !ok {
		return fmt.Errorf("role not extracted")
	}
	childRole, ok := c.Get(childRoleKey).(models.Role)
	if !ok {
		return fmt.Errorf("child role not extracted")
	}
	ctx := getContext(c)
	if edge, err := findRoleEdge(ctx, v.core, role.ID, childRole.ID); err != nil {
		return err
	} else if edge != nil {
		return errorResponse{
			Code: http.StatusBadRequest,
			Message: localize(
				c, "Role \"{role}\" already has child \"{child}\".",
				replaceField("role", role.Name),
				replaceField("child", childRole.Name),
			),
		}
	}
	edge := models.RoleEdge{
		RoleID:  role.ID,
		ChildID: childRole.ID,
	}
	if err := v.core.RoleEdges.Create(ctx, &edge); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, Role{
		ID:   childRole.ID,
		Name: childRole.Name,
	})
}

func (v *View) deleteRoleRole(c echo.Context) error {
	role, ok := c.Get(roleKey).(models.Role)
	if !ok {
		return fmt.Errorf("role not extracted")
	}
	childRole, ok := c.Get(childRoleKey).(models.Role)
	if !ok {
		return fmt.Errorf("child role not extracted")
	}
	if err := syncStore(c, v.core.RoleEdges); err != nil {
		return err
	}
	ctx := getContext(c)
	edge, err := findRoleEdge(ctx, v.core, role.ID, childRole.ID)
	if err != nil {
		return err
	}
	if edge == nil {
		return errorResponse{
			Code: http.StatusBadRequest,
			Message: localize(
				c, "Role \"{role}\" does not have child \"{child}\".",
				replaceField("role", role.Name),
				replaceField("child", childRole.Name),
			),
		}
	}
	if err := v.core.RoleEdges.Delete(ctx, edge.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, Role{
		ID:   childRole.ID,
		Name: childRole.Name,
	})
}

func (v *View) observeUserRoles(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return fmt.Errorf("user not extracted")
	}
	ctx := getContext(c)
	edges, err := v.core.AccountRoles.FindByAccount(ctx, user.AccountID)
	if err != nil {
		return err
	}
	defer func() { _ = edges.Close() }()
	var resp Roles
	for edges.Next() {
		edge := edges.Row()
		role, err := v.core.Roles.Get(ctx, edge.RoleID)
		if err != nil {
			if err == sql.ErrNoRows {
				c.Logger().Warnf("Role %v not found", edge.RoleID)
				continue
			}
			return err
		}
		resp.Roles = append(resp.Roles, Role{
			ID:   role.ID,
			Name: role.Name,
		})
	}
	if err := edges.Err(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) createUserRole(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return fmt.Errorf("user not extracted")
	}
	role, ok := c.Get(roleKey).(models.Role)
	if !ok {
		return fmt.Errorf("role not extracted")
	}
	ctx := getContext(c)
	if edge, err := findAccountRole(ctx, v.core, user.AccountID, role.ID); err != nil {
		return err
	} else if edge != nil {
		return errorResponse{
			Code: http.StatusBadRequest,
			Message: localize(
				c, "User \"{user}\" already has role \"{role}\".",
				replaceField("user", user.Login),
				replaceField("role", role.Name),
			),
		}
	}
	edge := models.AccountRole{
		AccountID: user.AccountID,
		RoleID:    role.ID,
	}
	if err := v.core.AccountRoles.Create(ctx, &edge); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, Role{
		ID:   role.ID,
		Name: role.Name,
	})
}

func (v *View) deleteUserRole(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return fmt.Errorf("user not extracted")
	}
	role, ok := c.Get(roleKey).(models.Role)
	if !ok {
		return fmt.Errorf("role not extracted")
	}
	if err := syncStore(c, v.core.AccountRoles); err != nil {
		return err
	}
	ctx := getContext(c)
	edge, err := findAccountRole(ctx, v.core, user.AccountID, role.ID)
	if err != nil {
		return err
	}
	if edge == nil {
		return errorResponse{
			Code: http.StatusBadRequest,
			Message: localize(
				c, "User \"{user}\" does not have role \"{role}\".",
				replaceField("user", user.Login),
				replaceField("role", role.Name),
			),
		}
	}
	if err := v.core.AccountRoles.Delete(ctx, edge.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, Role{
		ID:   role.ID,
		Name: role.Name,
	})
}

func findAccountRole(ctx context.Context, c *core.Core, accountID int64, roleID int64) (*models.AccountRole, error) {
	roles, err := c.AccountRoles.FindByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = roles.Close() }()
	for roles.Next() {
		role := roles.Row()
		if role.RoleID == roleID {
			return &role, nil
		}
	}
	return nil, roles.Err()
}

func findRoleEdge(ctx context.Context, c *core.Core, roleID int64, childID int64) (*models.RoleEdge, error) {
	roles, err := c.RoleEdges.FindByRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = roles.Close() }()
	for roles.Next() {
		role := roles.Row()
		if role.ChildID == childID {
			return &role, nil
		}
	}
	return nil, roles.Err()
}

func getRoleByParam(
	c echo.Context,
	roles *models.RoleStore,
	name string,
) (models.Role, error) {
	ctx := getContext(c)
	id, err := strconv.ParseInt(name, 10, 64)
	if err != nil {
		return roles.GetByName(ctx, name)
	}
	return roles.Get(ctx, id)
}

func (v *View) extractRole(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.Param("role")
		if err := syncStore(c, v.core.Roles); err != nil {
			return err
		}
		role, err := getRoleByParam(c, v.core.Roles, name)
		if err == sql.ErrNoRows {
			resp := errorResponse{
				Message: localize(
					c, "Role \"{role}\" not found.",
					replaceField("role", name),
				),
			}
			return c.JSON(http.StatusNotFound, resp)
		} else if err != nil {
			c.Logger().Error(err)
			return err
		}
		c.Set(roleKey, role)
		return next(c)
	}
}

func (v *View) extractChildRole(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.Param("child_role")
		if err := syncStore(c, v.core.Roles); err != nil {
			return err
		}
		role, err := getRoleByParam(c, v.core.Roles, name)
		if err == sql.ErrNoRows {
			resp := errorResponse{
				Message: localize(
					c, "Role \"{role}\" not found.",
					replaceField("role", name),
				),
			}
			return c.JSON(http.StatusNotFound, resp)
		} else if err != nil {
			c.Logger().Error(err)
			return err
		}
		c.Set(childRoleKey, role)
		return next(c)
	}
}

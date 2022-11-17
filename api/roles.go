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
	// Name contains role name.
	Name string `json:"name"`
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
		v.requirePermission(models.ObserveRolesRole),
	)
	g.POST(
		"/v0/roles", v.createRole,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(models.CreateRoleRole),
	)
	g.DELETE(
		"/v0/roles/:role", v.deleteRole,
		v.extractAuth(v.sessionAuth), v.extractRole,
		v.requirePermission(models.DeleteRoleRole),
	)
	g.GET(
		"/v0/roles/:role/roles", v.observeRoleRoles,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractRole,
		v.requirePermission(models.ObserveRoleRolesRole),
	)
	g.POST(
		"/v0/roles/:role/roles/:child_role", v.createRoleRole,
		v.extractAuth(v.sessionAuth), v.extractRole, v.extractChildRole,
		v.requirePermission(models.CreateRoleRoleRole),
	)
	g.DELETE(
		"/v0/roles/:role/roles/:child_role", v.deleteRoleRole,
		v.extractAuth(v.sessionAuth), v.extractRole, v.extractChildRole,
		v.requirePermission(models.DeleteRoleRoleRole),
	)
	g.GET(
		"/v0/users/:user/roles", v.observeUserRoles,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractUser,
		v.requirePermission(models.ObserveUserRolesRole),
	)
	g.POST(
		"/v0/users/:user/roles/:role", v.createUserRole,
		v.extractAuth(v.sessionAuth), v.extractUser, v.extractRole,
		v.requirePermission(models.CreateUserRoleRole),
	)
	g.DELETE(
		"/v0/users/:user/roles/:role", v.deleteUserRole,
		v.extractAuth(v.sessionAuth), v.extractUser, v.extractRole,
		v.requirePermission(models.DeleteUserRoleRole),
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
		return err
	}
	for _, role := range roles {
		resp.Roles = append(resp.Roles, Role{
			ID:   role.ID,
			Name: role.Name,
		})
	}
	sortFunc(resp.Roles, roleGreater)
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
	} else if len(f.Name) > 32 {
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
	if _, err := roles.GetByName(role.Name); err != sql.ErrNoRows {
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
	if role.IsBuiltIn() {
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
	edges, err := v.core.RoleEdges.FindByRole(role.ID)
	if err != nil {
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
			return err
		}
		resp.Roles = append(resp.Roles, Role{
			ID:   role.ID,
			Name: role.Name,
		})
	}
	sortFunc(resp.Roles, roleGreater)
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
	edges, err := v.core.RoleEdges.FindByRole(role.ID)
	if err != nil {
		return err
	}
	for _, edge := range edges {
		if edge.ChildID == childRole.ID {
			return errorResponse{
				Code: http.StatusBadRequest,
				Message: localize(
					c, "Role \"{role}\" already has child \"{child}\".",
					replaceField("role", role.Name),
					replaceField("child", childRole.Name),
				),
			}
		}
	}
	edge := models.RoleEdge{
		RoleID:  role.ID,
		ChildID: childRole.ID,
	}
	if err := v.core.RoleEdges.Create(getContext(c), &edge); err != nil {
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
	edges, err := v.core.RoleEdges.FindByRole(role.ID)
	if err != nil {
		return err
	}
	edgePos := -1
	for i, edge := range edges {
		if edge.ChildID == childRole.ID {
			edgePos = i
			break
		}
	}
	if edgePos == -1 {
		return errorResponse{
			Code: http.StatusBadRequest,
			Message: localize(
				c, "Role \"{role}\" does not have child \"{child}\".",
				replaceField("role", role.Name),
				replaceField("child", childRole.Name),
			),
		}
	}
	edge := edges[edgePos]
	if err := v.core.RoleEdges.Delete(getContext(c), edge.ID); err != nil {
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
	edges, err := v.core.AccountRoles.FindByAccount(user.AccountID)
	if err != nil {
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
			return err
		}
		resp.Roles = append(resp.Roles, Role{
			ID:   role.ID,
			Name: role.Name,
		})
	}
	sortFunc(resp.Roles, roleGreater)
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
	edges, err := v.core.AccountRoles.FindByAccount(user.AccountID)
	if err != nil {
		return err
	}
	for _, edge := range edges {
		if edge.RoleID == role.ID {
			return errorResponse{
				Code: http.StatusBadRequest,
				Message: localize(
					c, "User \"{user}\" already has role \"{role}\".",
					replaceField("user", user.Login),
					replaceField("role", role.Name),
				),
			}
		}
	}
	edge := models.AccountRole{
		AccountID: user.AccountID,
		RoleID:    role.ID,
	}
	if err := v.core.AccountRoles.Create(getContext(c), &edge); err != nil {
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
	edges, err := v.core.AccountRoles.FindByAccount(user.AccountID)
	if err != nil {
		return err
	}
	edgePos := -1
	for i, edge := range edges {
		if edge.RoleID == role.ID {
			edgePos = i
			break
		}
	}
	if edgePos == -1 {
		return errorResponse{
			Code: http.StatusBadRequest,
			Message: localize(
				c, "User \"{user}\" does not have role \"{role}\".",
				replaceField("user", user.Login),
				replaceField("role", role.Name),
			),
		}
	}
	edge := edges[edgePos]
	if err := v.core.AccountRoles.Delete(getContext(c), edge.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, Role{
		ID:   role.ID,
		Name: role.Name,
	})
}

func getRoleByParam(roles *models.RoleStore, name string) (models.Role, error) {
	id, err := strconv.ParseInt(name, 10, 64)
	if err != nil {
		return roles.GetByName(name)
	}
	return roles.Get(id)
}

func (v *View) extractRole(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.Param("role")
		if err := syncStore(c, v.core.Roles); err != nil {
			return err
		}
		role, err := getRoleByParam(v.core.Roles, name)
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
		role, err := getRoleByParam(v.core.Roles, name)
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

func roleGreater(l, r Role) bool {
	return l.ID > r.ID
}

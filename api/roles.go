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
		v.sessionAuth,
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
		v.sessionAuth, v.extractRole,
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
		v.sessionAuth, v.extractUser,
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
			Name: role.Name,
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

func (f createRoleForm) validate() *errorResponse {
	errors := errorFields{}
	if len(f.Name) < 3 {
		errors["name"] = errorField{Message: "name too short (<3)"}
	} else if len(f.Name) > 32 {
		errors["name"] = errorField{Message: "name too long (>32)"}
	} else if !roleNameRegexp.MatchString(f.Name) {
		errors["name"] = errorField{Message: "name has invalid format"}
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
	role.Name = f.Name
	if _, err := roles.GetByName(role.Name); err != sql.ErrNoRows {
		if err != nil {
			return &errorResponse{Message: "unknown error"}
		}
		return &errorResponse{
			Message: fmt.Sprintf("role %q already exists", role.Name),
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
	if err := v.core.Roles.Create(c.Request().Context(), &role); err != nil {
		c.Logger().Error(err)
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
		c.Logger().Error("role not extracted")
		return fmt.Errorf("role not extracted")
	}
	if role.IsBuiltIn() {
		return c.JSON(http.StatusBadRequest, errorResponse{
			Message: "unable to delete builtin role",
		})
	}
	if err := v.core.Roles.Delete(c.Request().Context(), role.ID); err != nil {
		c.Logger().Error(err)
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
			Name: role.Name,
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
					role.Name, childRole.Name,
				),
			})
		}
		role, err := v.core.Roles.Get(edge.ChildID)
		if err != nil {
			c.Logger().Error(err)
		} else {
			resp.Roles = append(resp.Roles, Role{
				ID:   role.ID,
				Name: role.Name,
			})
		}
	}
	edge := models.RoleEdge{
		RoleID:  role.ID,
		ChildID: childRole.ID,
	}
	if err := v.core.RoleEdges.Create(c.Request().Context(), &edge); err != nil {
		c.Logger().Error(err)
		return err
	}
	resp.Roles = append(resp.Roles, Role{
		ID:   childRole.ID,
		Name: childRole.Name,
	})
	return c.JSON(http.StatusCreated, resp)
}

func (v *View) deleteRoleRole(c echo.Context) error {
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
	edgePos := -1
	var resp Roles
	for i, edge := range edges {
		if edgePos == -1 && edge.ChildID == childRole.ID {
			edgePos = i
			continue
		}
		role, err := v.core.Roles.Get(edge.ChildID)
		if err != nil {
			c.Logger().Error(err)
		} else {
			resp.Roles = append(resp.Roles, Role{
				ID:   role.ID,
				Name: role.Name,
			})
		}
	}
	if edgePos == -1 {
		return c.JSON(http.StatusBadRequest, &errorResponse{
			Message: fmt.Sprintf(
				"role %q does not have child %q",
				role.Name, childRole.Name,
			),
		})
	}
	if err := v.core.RoleEdges.Delete(
		c.Request().Context(), edges[edgePos].ID,
	); err != nil {
		c.Logger().Error(err)
		return err
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
			Name: role.Name,
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
					user.Login, role.Name,
				),
			})
		}
		role, err := v.core.Roles.Get(edge.RoleID)
		if err != nil {
			c.Logger().Error(err)
		} else {
			resp.Roles = append(resp.Roles, Role{
				ID:   role.ID,
				Name: role.Name,
			})
		}
	}
	edge := models.AccountRole{
		AccountID: user.AccountID,
		RoleID:    role.ID,
	}
	if err := v.core.AccountRoles.Create(c.Request().Context(), &edge); err != nil {
		c.Logger().Error(err)
		return err
	}
	resp.Roles = append(resp.Roles, Role{
		ID:   role.ID,
		Name: role.Name,
	})
	return c.JSON(http.StatusCreated, resp)
}

func (v *View) deleteUserRole(c echo.Context) error {
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
	edgePos := -1
	var resp Roles
	for i, edge := range edges {
		if edgePos == -1 && edge.RoleID == role.ID {
			edgePos = i
			continue
		}
		role, err := v.core.Roles.Get(edge.RoleID)
		if err != nil {
			c.Logger().Error(err)
		} else {
			resp.Roles = append(resp.Roles, Role{
				ID:   role.ID,
				Name: role.Name,
			})
		}
	}
	if edgePos == -1 {
		return c.JSON(http.StatusBadRequest, &errorResponse{
			Message: fmt.Sprintf(
				"user %q does not have role %q",
				user.Login, role.Name,
			),
		})
	}
	if err := v.core.AccountRoles.Delete(
		c.Request().Context(), edges[edgePos].ID,
	); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, resp)
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
		role, err := getRoleByParam(v.core.Roles, name)
		if err == sql.ErrNoRows {
			resp := errorResponse{
				Message: fmt.Sprintf("role %q not found", name),
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
		role, err := getRoleByParam(v.core.Roles, name)
		if err == sql.ErrNoRows {
			resp := errorResponse{
				Message: fmt.Sprintf("role %q not found", name),
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

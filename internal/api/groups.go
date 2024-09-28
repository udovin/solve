package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
)

// registerGroupHandlers registers handlers for group management.
func (v *View) registerGroupHandlers(g *echo.Group) {
	g.GET(
		"/v0/groups", v.observeGroups,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(perms.ObserveGroupsRole),
	)
	g.GET(
		"/v0/groups/:group", v.observeGroup,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractGroup,
		v.requirePermission(perms.ObserveGroupRole),
	)
	g.POST(
		"/v0/groups", v.createGroup,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(perms.CreateGroupRole),
	)
	g.PATCH(
		"/v0/groups/:group", v.updateGroup,
		v.extractAuth(v.sessionAuth), v.extractGroup,
		v.requirePermission(perms.UpdateGroupRole),
	)
	g.DELETE(
		"/v0/groups/:group", v.deleteGroup,
		v.extractAuth(v.sessionAuth), v.extractGroup,
		v.requirePermission(perms.DeleteGroupRole),
	)
}

func (v *View) observeGroups(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	if err := syncStore(c, v.core.Groups); err != nil {
		return err
	}
	var resp Groups
	groups, err := v.core.Groups.ReverseAll(getContext(c), 0, 0)
	if err != nil {
		return err
	}
	defer func() { _ = groups.Close() }()
	for groups.Next() {
		group := groups.Row()
		permissions := v.getGroupPermissions(accountCtx, group)
		if permissions.HasPermission(perms.ObserveGroupRole) {
			resp.Groups = append(resp.Groups, makeGroup(group, permissions))
		}
	}
	if err := groups.Err(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeGroup(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	group, ok := c.Get(groupKey).(models.Group)
	if !ok {
		return fmt.Errorf("group not extracted")
	}
	permissions := v.getGroupPermissions(accountCtx, group)
	return c.JSON(http.StatusOK, makeGroup(group, permissions))
}

type UpdateGroupForm struct {
	Title   *string `json:"title"`
	OwnerID *int64  `json:"owner_id"`
}

func (f *UpdateGroupForm) Update(c echo.Context, o *models.Group) error {
	errors := errorFields{}
	if f.Title != nil {
		if len(*f.Title) < 4 {
			errors["title"] = errorField{
				Message: localize(c, "Title is too short."),
			}
		} else if len(*f.Title) > 64 {
			errors["title"] = errorField{
				Message: localize(c, "Title is too long."),
			}
		}
		o.Title = *f.Title
	}
	if len(errors) > 0 {
		return &errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	return nil
}

type CreateGroupForm UpdateGroupForm

func (f *CreateGroupForm) Update(c echo.Context, o *models.Group) error {
	if f.Title == nil {
		return &errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Form has invalid fields."),
			InvalidFields: errorFields{
				"title": errorField{
					Message: localize(c, "Title is required."),
				},
			},
		}
	}
	return (*UpdateGroupForm)(f).Update(c, o)
}

func (v *View) createGroup(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	var form CreateGroupForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var group models.Group
	if err := form.Update(c, &group); err != nil {
		return err
	}
	if account := accountCtx.Account; account != nil {
		group.OwnerID = NInt64(account.ID)
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		account := models.Account{Kind: group.AccountKind()}
		if err := v.core.Accounts.Create(ctx, &account); err != nil {
			return err
		}
		group.ID = account.ID
		return v.core.Groups.Create(ctx, &group)
	}, sqlRepeatableRead); err != nil {
		c.Logger().Error(err)
		return err
	}
	permissions := v.getGroupPermissions(accountCtx, group)
	return c.JSON(http.StatusCreated, makeGroup(group, permissions))
}

func (v *View) updateGroup(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	group, ok := c.Get(groupKey).(models.Group)
	if !ok {
		return fmt.Errorf("group not extracted")
	}
	permissions := v.getGroupPermissions(accountCtx, group)
	var form UpdateGroupForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if err := form.Update(c, &group); err != nil {
		return err
	}
	var missingPermissions []string
	if form.OwnerID != nil {
		if !permissions.HasPermission(perms.UpdateGroupOwnerRole) {
			missingPermissions = append(missingPermissions, perms.UpdateGroupOwnerRole)
		} else {
			if _, err := v.core.Users.Get(getContext(c), *form.OwnerID); err != nil {
				if err == sql.ErrNoRows {
					return errorResponse{
						Code:    http.StatusBadRequest,
						Message: localize(c, "User not found."),
					}
				}
				return err
			}
			group.OwnerID = models.NInt64(*form.OwnerID)
		}
	}
	if len(missingPermissions) > 0 {
		return errorResponse{
			Code:               http.StatusForbidden,
			Message:            localize(c, "Account missing permissions."),
			MissingPermissions: missingPermissions,
		}
	}
	if err := v.core.Groups.Update(getContext(c), group); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, makeGroup(group, permissions))
}

func (v *View) deleteGroup(c echo.Context) error {
	group, ok := c.Get(groupKey).(models.Group)
	if !ok {
		return fmt.Errorf("group not extracted")
	}
	if err := v.core.Groups.Delete(getContext(c), group.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, makeGroup(group, perms.PermissionSet{}))
}

func (v *View) extractGroup(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
		if !ok {
			return fmt.Errorf("auth not extracted")
		}
		id, err := strconv.ParseInt(c.Param("group"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Invalid group ID."),
			}
		}
		if err := syncStore(c, v.core.Groups); err != nil {
			return err
		}
		group, err := v.core.Groups.Get(getContext(c), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: localize(c, "Group not found."),
				}
			}
			return err
		}
		c.Set(groupKey, group)
		c.Set(permissionCtxKey, v.getGroupPermissions(accountCtx, group))
		return next(c)
	}
}

func (v *View) getGroupPermissions(
	ctx *managers.AccountContext, group models.Group,
) perms.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if ctx.Account.ID != 0 && ctx.Account.ID == int64(group.OwnerID) {
		permissions.AddPermission(
			perms.ObserveGroupRole,
			perms.UpdateGroupRole,
			perms.UpdateGroupOwnerRole,
			perms.DeleteGroupRole,
		)
		for _, member := range ctx.GroupMembers {
			if member.GroupID != group.ID {
				continue
			}
			switch member.Kind {
			case models.RegularMember:
				permissions.AddPermission(
					perms.ObserveGroupRole,
				)
			case models.ManagerMember:
				permissions.AddPermission(
					perms.ObserveGroupRole,
					perms.UpdateGroupRole,
				)
			}
		}
	}
	return permissions
}

type Group struct {
	ID          int64    `json:"id"`
	Title       string   `json:"title"`
	Permissions []string `json:"permissions"`
}

type Groups struct {
	Groups []Group `json:"group"`
}

var groupPermissions = []string{
	perms.UpdateGroupRole,
	perms.UpdateGroupOwnerRole,
	perms.DeleteGroupRole,
}

func makeGroup(group models.Group, permissions perms.Permissions) Group {
	resp := Group{
		ID:    group.ID,
		Title: group.Title,
	}
	for _, permission := range groupPermissions {
		if permissions.HasPermission(permission) {
			resp.Permissions = append(resp.Permissions, permission)
		}
	}
	return resp
}

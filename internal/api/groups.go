package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/internal/core"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
	"github.com/udovin/solve/internal/pkg/logs"
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
	g.GET(
		"/v0/groups/:group/members", v.observeGroupMembers,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractGroup,
		v.requirePermission(perms.ObserveGroupMembersRole),
	)
	g.POST(
		"/v0/groups/:group/members", v.createGroupMember,
		v.extractAuth(v.sessionAuth), v.extractGroup,
		v.requirePermission(perms.CreateGroupMemberRole),
	)
	g.PATCH(
		"/v0/groups/:group/members/:member", v.updateGroupMember,
		v.extractAuth(v.sessionAuth), v.extractGroup, v.extractGroupMember,
		v.requirePermission(perms.UpdateGroupMemberRole),
	)
	g.DELETE(
		"/v0/groups/:group/members/:member", v.deleteGroupMember,
		v.extractAuth(v.sessionAuth), v.extractGroup, v.extractGroupMember,
		v.requirePermission(perms.DeleteGroupMemberRole),
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

func (v *View) observeGroupMembers(c echo.Context) error {
	group, ok := c.Get(groupKey).(models.Group)
	if !ok {
		return fmt.Errorf("group not extracted")
	}
	ctx := getContext(c)
	if err := syncStore(c, v.core.GroupMembers); err != nil {
		return err
	}
	members, err := v.core.GroupMembers.FindByGroup(ctx, group.ID)
	if err != nil {
		return err
	}
	defer func() { _ = members.Close() }()
	resp := GroupMembers{}
	for members.Next() {
		resp.Members = append(resp.Members, makeGroupMember(c, members.Row(), v.core))
	}
	if err := members.Err(); err != nil {
		return err
	}
	sortFunc(resp.Members, groupMemberLess)
	return c.JSON(http.StatusOK, resp)
}

type MemberKind = models.MemberKind

type CreateGroupMemberForm struct {
	AccountID int64      `json:"account_id"`
	Kind      MemberKind `json:"kind"`
}

func (f *CreateGroupMemberForm) Update(c echo.Context, o *models.GroupMember, core *core.Core) error {
	ctx := getContext(c)
	account, err := core.Accounts.Get(ctx, f.AccountID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "User not found."),
			}
		}
		return err
	}
	if account.Kind != models.UserAccountKind {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "User not found."),
		}
	}
	if !f.Kind.IsValid() {
		f.Kind = models.RegularMember
	}
	o.AccountID = f.AccountID
	o.Kind = f.Kind
	return nil
}

func (v *View) createGroupMember(c echo.Context) error {
	group, ok := c.Get(groupKey).(models.Group)
	if !ok {
		return fmt.Errorf("group not extracted")
	}
	var form CreateGroupMemberForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var member models.GroupMember
	if err := form.Update(c, &member, v.core); err != nil {
		return err
	}
	member.GroupID = group.ID
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		return v.core.GroupMembers.Create(ctx, &member)
	}, sqlRepeatableRead); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusCreated, makeGroupMember(c, member, v.core))
}

type UpdateGroupMemberForm struct {
	Kind MemberKind `json:"kind"`
}

func (f *UpdateGroupMemberForm) Update(c echo.Context, o *models.GroupMember) error {
	if !f.Kind.IsValid() {
		f.Kind = models.RegularMember
	}
	o.Kind = f.Kind
	return nil
}

func (v *View) updateGroupMember(c echo.Context) error {
	member, ok := c.Get(groupMemberKey).(models.GroupMember)
	if !ok {
		return fmt.Errorf("group member not extracted")
	}
	var form UpdateGroupMemberForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if err := form.Update(c, &member); err != nil {
		return err
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		return v.core.GroupMembers.Update(ctx, member)
	}, sqlRepeatableRead); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, makeGroupMember(c, member, v.core))
}

func (v *View) deleteGroupMember(c echo.Context) error {
	member, ok := c.Get(groupMemberKey).(models.GroupMember)
	if !ok {
		return fmt.Errorf("group member not extracted")
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		return v.core.GroupMembers.Delete(ctx, member.ID)
	}, sqlRepeatableRead); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, makeGroupMember(c, member, v.core))
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

func (v *View) extractGroupMember(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		group, ok := c.Get(groupKey).(models.Group)
		if !ok {
			return fmt.Errorf("group not extracted")
		}
		id, err := strconv.ParseInt(c.Param("member"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Invalid group member ID."),
			}
		}
		if err := syncStore(c, v.core.GroupMembers); err != nil {
			return err
		}
		member, err := v.core.GroupMembers.Get(getContext(c), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: localize(c, "Group member not found."),
				}
			}
			return err
		}
		if member.GroupID != group.ID {
			return errorResponse{
				Code:    http.StatusNotFound,
				Message: localize(c, "Group member not found."),
			}
		}
		c.Set(groupMemberKey, member)
		return next(c)
	}
}

func (v *View) getGroupPermissions(
	ctx *managers.AccountContext, group models.Group,
) perms.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if group.OwnerID != 0 && ctx.Account.ID == int64(group.OwnerID) {
		permissions.AddPermission(
			perms.ObserveGroupRole,
			perms.UpdateGroupRole,
			perms.UpdateGroupOwnerRole,
			perms.DeleteGroupRole,
			perms.ObserveGroupMembersRole,
			perms.CreateGroupMemberRole,
			perms.UpdateGroupMemberRole,
			perms.DeleteGroupMemberRole,
		)
	}
	for _, member := range ctx.GroupMembers {
		if member.GroupID != group.ID {
			continue
		}
		switch member.Kind {
		case models.RegularMember:
			permissions.AddPermission(
				perms.ObserveGroupRole,
				perms.ObserveGroupMembersRole,
			)
		case models.ManagerMember:
			permissions.AddPermission(
				perms.ObserveGroupRole,
				perms.UpdateGroupRole,
				perms.ObserveGroupMembersRole,
				perms.CreateGroupMemberRole,
				perms.UpdateGroupMemberRole,
				perms.DeleteGroupMemberRole,
			)
		}
	}
	return permissions
}

type Group struct {
	ID          int64    `json:"id"`
	Title       string   `json:"title"`
	Permissions []string `json:"permissions,omitempty"`
}

type Groups struct {
	Groups []Group `json:"groups"`
}

type GroupMember struct {
	ID      int64             `json:"id"`
	Kind    models.MemberKind `json:"kind"`
	Account *Account          `json:"account"`
}

type GroupMembers struct {
	Members []GroupMember `json:"members"`
}

var groupPermissions = []string{
	perms.UpdateGroupRole,
	perms.UpdateGroupOwnerRole,
	perms.DeleteGroupRole,
	perms.ObserveGroupMembersRole,
	perms.CreateGroupMemberRole,
	perms.UpdateGroupMemberRole,
	perms.DeleteGroupMemberRole,
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

func makeGroupMember(c echo.Context, member models.GroupMember, core *core.Core) GroupMember {
	resp := GroupMember{
		ID:   member.ID,
		Kind: member.Kind,
	}
	ctx := getContext(c)
	if account, err := core.Accounts.Get(ctx, member.AccountID); err == nil {
		switch account.Kind {
		case models.UserAccountKind:
			if user, err := core.Users.Get(ctx, account.ID); err == nil {
				resp.Account = &Account{
					ID:   account.ID,
					Kind: user.AccountKind(),
					User: &User{
						ID:    user.ID,
						Login: user.Login,
					},
				}
			}
		default:
			c.Logger().Warn(
				"Unsupported account kind",
				logs.Any("id", account.ID),
				logs.Any("kind", account.Kind),
			)
		}
	}
	return resp
}

func groupMemberLess(l, r GroupMember) bool {
	return l.ID < r.ID
}

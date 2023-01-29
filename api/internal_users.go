package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

// registerInternalUserHandlers registers handlers for internal user management.
func (v *View) registerInternalUserHandlers(g *echo.Group) {
	g.GET(
		"/v0/internal-groups", v.observeInternalGroups,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(models.ObserveInternalGroupsRole),
	)
	g.POST(
		"/v0/internal-groups", v.createInternalGroup,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(models.CreateInternalGroupRole),
	)
	g.PATCH(
		"/v0/internal-groups/:group", v.updateInternalGroup,
		v.extractAuth(v.sessionAuth), v.extractInternalGroup,
		v.requirePermission(models.UpdateInternalGroupRole),
	)
	g.DELETE(
		"/v0/internal-groups/:group", v.deleteInternalGroup,
		v.extractAuth(v.sessionAuth), v.extractInternalGroup,
		v.requirePermission(models.DeleteInternalGroupRole),
	)
}

func (v *View) observeInternalGroups(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	if err := syncStore(c, v.core.InternalGroups); err != nil {
		return err
	}
	var resp InternalGroups
	groups, err := v.core.InternalGroups.All()
	if err != nil {
		return err
	}
	for _, group := range groups {
		permissions := v.getInternalGroupPermissions(accountCtx, group)
		if permissions.HasPermission(models.ObserveInternalGroupRole) {
			resp.Groups = append(
				resp.Groups,
				makeInternalGroup(group),
			)
		}
	}
	sortFunc(resp.Groups, internalGroupGreater)
	return c.JSON(http.StatusOK, resp)
}

type updateInternalGroupForm struct {
	Title *string `json:"title"`
}

func (f *updateInternalGroupForm) Update(c echo.Context, o *models.InternalGroup) error {
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

type createInternalGroupForm updateInternalGroupForm

func (f *createInternalGroupForm) Update(c echo.Context, o *models.InternalGroup) error {
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
	return (*updateInternalGroupForm)(f).Update(c, o)
}

func (v *View) createInternalGroup(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	var form createInternalGroupForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var group models.InternalGroup
	if err := form.Update(c, &group); err != nil {
		return err
	}
	if account := accountCtx.Account; account != nil {
		group.OwnerID = NInt64(account.ID)
	}
	if err := v.core.InternalGroups.Create(getContext(c), &group); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, makeInternalGroup(group))
}

func (v *View) updateInternalGroup(c echo.Context) error {
	group, ok := c.Get(internalGroupKey).(models.InternalGroup)
	if !ok {
		return fmt.Errorf("internal group not extracted")
	}
	var form updateInternalGroupForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if err := form.Update(c, &group); err != nil {
		return err
	}
	if err := v.core.InternalGroups.Update(getContext(c), group); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, makeInternalGroup(group))
}

func (v *View) deleteInternalGroup(c echo.Context) error {
	group, ok := c.Get(internalGroupKey).(models.InternalGroup)
	if !ok {
		return fmt.Errorf("internal group not extracted")
	}
	if err := v.core.InternalGroups.Delete(getContext(c), group.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, makeInternalGroup(group))
}

func (v *View) extractInternalGroup(next echo.HandlerFunc) echo.HandlerFunc {
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
				Message: localize(c, "Invalid internal group ID."),
			}
		}
		if err := syncStore(c, v.core.InternalGroups); err != nil {
			return err
		}
		group, err := v.core.InternalGroups.Get(id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: localize(c, "Internal group not found."),
				}
			}
			return err
		}
		c.Set(internalGroupKey, group)
		c.Set(permissionCtxKey, v.getInternalGroupPermissions(accountCtx, group))
		return next(c)
	}
}

func (v *View) getInternalGroupPermissions(
	ctx *managers.AccountContext, group models.InternalGroup,
) managers.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if ctx.Account.ID != 0 && ctx.Account.ID == int64(group.OwnerID) {
		permissions.AddPermission(
			models.ObserveInternalGroupRole,
			models.UpdateInternalGroupRole,
			models.DeleteInternalGroupRole,
			models.ObserveInternalUserRole,
			models.CreateInternalUserRole,
			models.UpdateInternalUserRole,
			models.DeleteInternalUserRole,
		)
	}
	return permissions
}

type InternalGroup struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type InternalGroups struct {
	Groups []InternalGroup `json:"groups"`
}

func makeInternalGroup(group models.InternalGroup) InternalGroup {
	return InternalGroup{
		ID:    group.ID,
		Title: group.Title,
	}
}

type InternalUser struct {
	ID       int64  `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password,omitempty"`
	Title    string `json:"title,omitempty"`
}

func makeInternalUser(user models.InternalUser) InternalUser {
	return InternalUser{
		ID:    user.ID,
		Login: user.Login,
		Title: string(user.Title),
	}
}

func internalGroupGreater(l, r InternalGroup) bool {
	return l.ID > r.ID
}

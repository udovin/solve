package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

// registerScopeHandlers registers handlers for scope management.
func (v *View) registerScopeHandlers(g *echo.Group) {
	g.GET(
		"/v0/scopes", v.observeScopes,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(models.ObserveScopesRole),
	)
	g.GET(
		"/v0/scopes/:scope", v.observeScope,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractScope,
		v.requirePermission(models.ObserveScopeRole),
	)
	g.POST(
		"/v0/scopes", v.createScope,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(models.CreateScopeRole),
	)
	g.PATCH(
		"/v0/scopes/:scope", v.updateScope,
		v.extractAuth(v.sessionAuth), v.extractScope,
		v.requirePermission(models.UpdateScopeRole),
	)
	g.DELETE(
		"/v0/scopes/:scope", v.deleteScope,
		v.extractAuth(v.sessionAuth), v.extractScope,
		v.requirePermission(models.DeleteScopeRole),
	)
	g.GET(
		"/v0/scopes/:scope/users", v.observeScopeUsers,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractScope,
		v.requirePermission(models.ObserveScopeRole),
	)
	g.POST(
		"/v0/scopes/:scope/users", v.createScopeUser,
		v.extractAuth(v.sessionAuth), v.extractScope,
		v.requirePermission(models.CreateScopeUserRole),
	)
	g.GET(
		"/v0/scopes/:scope/users/:user", v.observeScopeUser,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractScope, v.extractScopeUser,
		v.requirePermission(models.ObserveScopeUserRole),
	)
	g.PATCH(
		"/v0/scopes/:scope/users/:user", v.updateScopeUser,
		v.extractAuth(v.sessionAuth), v.extractScope, v.extractScopeUser,
		v.requirePermission(models.UpdateScopeUserRole),
	)
	g.POST(
		"/v0/scopes/:scope/users/:user/logout", v.logoutScopeUser,
		v.extractAuth(v.sessionAuth), v.extractScope, v.extractScopeUser,
		v.requirePermission(models.UpdateScopeUserRole),
	)
	g.DELETE(
		"/v0/scopes/:scope/users/:user", v.deleteScopeUser,
		v.extractAuth(v.sessionAuth), v.extractScope, v.extractScopeUser,
		v.requirePermission(models.DeleteScopeUserRole),
	)
}

func (v *View) observeScopes(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	if err := syncStore(c, v.core.Scopes); err != nil {
		return err
	}
	var resp Scopes
	scopes, err := v.core.Scopes.ReverseAll(getContext(c), 0)
	if err != nil {
		return err
	}
	defer func() { _ = scopes.Close() }()
	for scopes.Next() {
		scope := scopes.Row()
		permissions := v.getScopePermissions(accountCtx, scope)
		if permissions.HasPermission(models.ObserveScopeRole) {
			resp.Scopes = append(
				resp.Scopes,
				makeScope(scope),
			)
		}
	}
	if err := scopes.Err(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeScope(c echo.Context) error {
	scope, ok := c.Get(scopeKey).(models.Scope)
	if !ok {
		return fmt.Errorf("scope not extracted")
	}
	return c.JSON(http.StatusOK, makeScope(scope))
}

type updateScopeForm struct {
	Title *string `json:"title"`
}

func (f *updateScopeForm) Update(c echo.Context, o *models.Scope) error {
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

type createScopeForm updateScopeForm

func (f *createScopeForm) Update(c echo.Context, o *models.Scope) error {
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
	return (*updateScopeForm)(f).Update(c, o)
}

func (v *View) createScope(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		return fmt.Errorf("account not extracted")
	}
	var form createScopeForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var scope models.Scope
	if err := form.Update(c, &scope); err != nil {
		return err
	}
	if account := accountCtx.Account; account != nil {
		scope.OwnerID = NInt64(account.ID)
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		account := models.Account{Kind: scope.AccountKind()}
		if err := v.core.Accounts.Create(ctx, &account); err != nil {
			return err
		}
		scope.AccountID = account.ID
		return v.core.Scopes.Create(ctx, &scope)
	}, sqlRepeatableRead); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusCreated, makeScope(scope))
}

func (v *View) updateScope(c echo.Context) error {
	scope, ok := c.Get(scopeKey).(models.Scope)
	if !ok {
		return fmt.Errorf("scope not extracted")
	}
	var form updateScopeForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if err := form.Update(c, &scope); err != nil {
		return err
	}
	if err := v.core.Scopes.Update(getContext(c), scope); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, makeScope(scope))
}

func (v *View) deleteScope(c echo.Context) error {
	scope, ok := c.Get(scopeKey).(models.Scope)
	if !ok {
		return fmt.Errorf("scope not extracted")
	}
	if err := v.core.Scopes.Delete(getContext(c), scope.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, makeScope(scope))
}

func (v *View) observeScopeUsers(c echo.Context) error {
	scope, ok := c.Get(scopeKey).(models.Scope)
	if !ok {
		return fmt.Errorf("scope not extracted")
	}
	permissions, ok := c.Get(permissionCtxKey).(managers.Permissions)
	if !ok {
		return fmt.Errorf("permissions not extracted")
	}
	users, err := v.core.ScopeUsers.FindByScope(scope.ID)
	if err != nil {
		return err
	}
	resp := ScopeUsers{}
	for _, user := range users {
		if permissions.HasPermission(models.ObserveScopeUserRole) {
			resp.Users = append(resp.Users, makeScopeUser(user))
		}
	}
	sortFunc(resp.Users, scopeUserLess)
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeScopeUser(c echo.Context) error {
	user, ok := c.Get(scopeUserKey).(models.ScopeUser)
	if !ok {
		return fmt.Errorf("user not extracted")
	}
	return c.JSON(http.StatusOK, makeScopeUser(user))
}

type updateScopeUserForm struct {
	Login            *string `json:"login"`
	Title            *string `json:"title"`
	Password         *string `json:"password"`
	GeneratePassword *bool   `json:"generate_password"`
}

func (f *updateScopeUserForm) Update(
	c echo.Context, o *models.ScopeUser, users *models.ScopeUserStore,
) error {
	errors := errorFields{}
	if f.Login != nil {
		validateLogin(c, errors, *f.Login)
		o.Login = *f.Login
	}
	if f.Title != nil {
		title := []rune(*f.Title)
		if len(title) != 0 && len(title) < 4 {
			errors["title"] = errorField{
				Message: localize(c, "Title is too short."),
			}
		} else if len(title) > 64 {
			errors["title"] = errorField{
				Message: localize(c, "Title is too long."),
			}
		}
		o.Title = models.NString(*f.Title)
	}
	if f.GeneratePassword != nil && *f.GeneratePassword {
		password, err := generatePassword()
		if err != nil {
			return err
		}
		f.Password = &password
	}
	if f.Password != nil {
		if len(*f.Password) != 0 {
			validatePassword(c, errors, *f.Password)
			if err := users.SetPassword(o, *f.Password); err != nil {
				return err
			}
		} else {
			// Reset password.
			o.PasswordHash = ""
			o.PasswordSalt = ""
		}
	}
	if len(errors) > 0 {
		return errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	return nil
}

type createScopeUserForm updateScopeUserForm

func (f *createScopeUserForm) Update(
	c echo.Context, o *models.ScopeUser, users *models.ScopeUserStore,
) error {
	if f.Login == nil {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Form has invalid fields."),
			InvalidFields: errorFields{
				"login": errorField{Message: localize(c, "Login too short.")},
			},
		}
	}
	if f.Title == nil {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Form has invalid fields."),
			InvalidFields: errorFields{
				"title": errorField{Message: localize(c, "Title is required.")},
			},
		}
	}
	if f.GeneratePassword == nil && f.Password == nil {
		f.GeneratePassword = getPtr(true)
	}
	return (*updateScopeUserForm)(f).Update(c, o, users)
}

func (v *View) createScopeUser(c echo.Context) error {
	scope, ok := c.Get(scopeKey).(models.Scope)
	if !ok {
		return fmt.Errorf("scope not extracted")
	}
	var form createScopeUserForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var user models.ScopeUser
	if err := form.Update(c, &user, v.core.ScopeUsers); err != nil {
		return err
	}
	user.ScopeID = scope.ID
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		account := models.Account{Kind: user.AccountKind()}
		if err := v.core.Accounts.Create(ctx, &account); err != nil {
			return err
		}
		user.AccountID = account.ID
		return v.core.ScopeUsers.Create(ctx, &user)
	}, sqlRepeatableRead); err != nil {
		c.Logger().Error(err)
		return err
	}
	resp := makeScopeUser(user)
	if form.Password != nil {
		resp.Password = *form.Password
	}
	return c.JSON(http.StatusCreated, resp)
}

func (v *View) updateScopeUser(c echo.Context) error {
	user, ok := c.Get(scopeUserKey).(models.ScopeUser)
	if !ok {
		return fmt.Errorf("user not extracted")
	}
	var form updateScopeUserForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if err := form.Update(c, &user, v.core.ScopeUsers); err != nil {
		return err
	}
	if err := v.core.ScopeUsers.Update(getContext(c), user); err != nil {
		return err
	}
	resp := makeScopeUser(user)
	if form.Password != nil {
		resp.Password = *form.Password
	}
	return c.JSON(http.StatusCreated, resp)
}

func (v *View) logoutScopeUser(c echo.Context) error {
	user, ok := c.Get(scopeUserKey).(models.ScopeUser)
	if !ok {
		return fmt.Errorf("user not extracted")
	}
	sessions, err := v.core.Sessions.FindByAccount(user.AccountID)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if err := v.core.Sessions.Delete(getContext(c), session.ID); err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, makeScopeUser(user))
}

func (v *View) deleteScopeUser(c echo.Context) error {
	user, ok := c.Get(scopeUserKey).(models.ScopeUser)
	if !ok {
		return fmt.Errorf("user not extracted")
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		if err := v.core.ScopeUsers.Delete(ctx, user.ID); err != nil {
			return err
		}
		return v.core.Accounts.Delete(ctx, user.AccountID)
	}, sqlRepeatableRead); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, makeScopeUser(user))
}

func (v *View) extractScope(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
		if !ok {
			return fmt.Errorf("auth not extracted")
		}
		id, err := strconv.ParseInt(c.Param("scope"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Invalid scope ID."),
			}
		}
		if err := syncStore(c, v.core.Scopes); err != nil {
			return err
		}
		scope, err := v.core.Scopes.Get(getContext(c), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: localize(c, "Scope not found."),
				}
			}
			return err
		}
		c.Set(scopeKey, scope)
		c.Set(permissionCtxKey, v.getScopePermissions(accountCtx, scope))
		return next(c)
	}
}

func (v *View) extractScopeUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		scope, ok := c.Get(scopeKey).(models.Scope)
		if !ok {
			return fmt.Errorf("scope not extracted")
		}
		id, err := strconv.ParseInt(c.Param("user"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return errorResponse{
				Code:    http.StatusBadRequest,
				Message: localize(c, "Invalid user ID."),
			}
		}
		if err := syncStore(c, v.core.Scopes); err != nil {
			return err
		}
		user, err := v.core.ScopeUsers.Get(getContext(c), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: localize(c, "User not found."),
				}
			}
			return err
		}
		if user.ScopeID != scope.ID {
			return errorResponse{
				Code:    http.StatusNotFound,
				Message: localize(c, "User not found."),
			}
		}
		c.Set(scopeUserKey, user)
		return next(c)
	}
}

func (v *View) getScopePermissions(
	ctx *managers.AccountContext, scope models.Scope,
) managers.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if ctx.Account.ID != 0 && ctx.Account.ID == int64(scope.OwnerID) {
		permissions.AddPermission(
			models.ObserveScopeRole,
			models.UpdateScopeRole,
			models.DeleteScopeRole,
			models.ObserveScopeUserRole,
			models.CreateScopeUserRole,
			models.UpdateScopeUserRole,
			models.DeleteScopeUserRole,
		)
	}
	return permissions
}

type Scope struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type Scopes struct {
	Scopes []Scope `json:"scopes"`
}

func makeScope(scope models.Scope) Scope {
	return Scope{
		ID:    scope.ID,
		Title: scope.Title,
	}
}

type ScopeUser struct {
	ID       int64  `json:"id"`
	Login    string `json:"login"`
	Title    string `json:"title,omitempty"`
	Password string `json:"password,omitempty"`
}

type ScopeUsers struct {
	Users []ScopeUser `json:"users"`
}

func makeScopeUser(user models.ScopeUser) ScopeUser {
	return ScopeUser{
		ID:    user.ID,
		Login: user.Login,
		Title: string(user.Title),
	}
}

func generatePassword() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func scopeUserLess(l, r ScopeUser) bool {
	return l.ID < r.ID
}

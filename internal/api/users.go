package api

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/internal/config"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
	"github.com/udovin/solve/internal/pkg/logs"
)

// User represents user.
type User struct {
	// ID contains user ID.
	ID int64 `json:"id"`
	// Login contains user login.
	Login string `json:"login"`
	// Email contains user email.
	Email string `json:"email,omitempty"`
	// Status contains user status.
	Status string `json:"status,omitempty"`
	// FirstName contains first name.
	FirstName string `json:"first_name,omitempty"`
	// LastName contains last name.
	LastName string `json:"last_name,omitempty"`
	// MiddleName contains middle name.
	MiddleName string `json:"middle_name,omitempty"`
	// UnconfirmedEmail contains email address that currently unconfirmed.
	UnconfirmedEmail string `json:"unconfirmed_email,omitempty"`
}

// Status represents current authorization status.
type Status struct {
	User        *User      `json:"user,omitempty"`
	ScopeUser   *ScopeUser `json:"scope_user,omitempty"`
	Session     *Session   `json:"session,omitempty"`
	Permissions []string   `json:"permissions"`
	Locale      string     `json:"locale,omitempty"`
}

// registerUserHandlers registers handlers for user management.
func (v *View) registerUserHandlers(g *echo.Group) {
	g.GET(
		"/v0/users/:user", v.observeUser,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractUser,
		v.requirePermission(perms.ObserveUserRole),
	)
	g.PATCH(
		"/v0/users/:user", v.updateUser,
		v.extractAuth(v.sessionAuth), v.extractUser,
		v.requirePermission(perms.UpdateUserRole),
	)
	g.GET(
		"/v0/users/:user/sessions", v.observeUserSessions,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractUser,
		v.requirePermission(perms.ObserveUserSessionsRole),
	)
	g.POST(
		"/v0/users/:user/status", v.updateUserStatus,
		v.extractAuth(v.sessionAuth), v.extractUser,
		v.requirePermission(perms.UpdateUserRole, perms.UpdateUserStatusRole),
	)
	g.POST(
		"/v0/users/:user/password", v.updateUserPassword,
		v.extractAuth(v.sessionAuth), v.extractUser,
		v.requirePermission(perms.UpdateUserRole, perms.UpdateUserPasswordRole),
	)
	g.POST(
		"/v0/users/:user/email", v.updateUserEmail,
		v.extractAuth(v.sessionAuth), v.extractUser,
		v.requirePermission(perms.UpdateUserRole, perms.UpdateUserEmailRole),
	)
	g.POST(
		"/v0/users/:user/email-resend", v.resendUserEmail,
		v.extractAuth(v.sessionAuth), v.extractUser,
		v.requirePermission(perms.UpdateUserRole, perms.UpdateUserEmailRole),
	)
	g.GET(
		"/v0/status", v.status,
		v.extractAuth(v.sessionAuth, v.scopeUserAuth, v.userAuth, v.guestAuth),
		v.requirePermission(perms.StatusRole),
	)
	g.POST(
		"/v0/login", v.loginAccount,
		v.extractAuth(v.scopeUserAuth, v.userAuth),
		v.requirePermission(perms.LoginRole),
	)
	g.POST(
		"/v0/logout", v.logoutAccount,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(perms.LogoutRole),
	)
	g.POST(
		"/v0/register", v.registerUser,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(perms.RegisterRole),
	)
	g.POST(
		"/v0/password-reset", v.resetUserPassword,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(perms.ResetPasswordRole),
	)
}

func (v *View) registerSocketUserHandlers(g *echo.Group) {
	if v.core.Users == nil {
		return
	}
	g.GET(
		"/v0/users/:user", v.observeUser, v.extractUser,
	)
	g.POST(
		"/v0/register", v.registerUser,
	)
}

func makeUser(user models.User, permissions perms.Permissions) User {
	assign := func(field *string, value, permission string) {
		if permissions.HasPermission(permission) {
			*field = value
		}
	}
	resp := User{ID: user.ID, Login: user.Login}
	assign(&resp.Status, user.Status.String(), perms.ObserveUserStatusRole)
	assign(&resp.Email, string(user.Email), perms.ObserveUserEmailRole)
	assign(&resp.FirstName, string(user.FirstName), perms.ObserveUserFirstNameRole)
	assign(&resp.LastName, string(user.LastName), perms.ObserveUserLastNameRole)
	assign(&resp.MiddleName, string(user.MiddleName), perms.ObserveUserMiddleNameRole)
	return resp
}

func (v *View) observeUser(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		c.Logger().Error("user not extracted")
		return fmt.Errorf("user not extracted")
	}
	permissions, ok := c.Get(permissionCtxKey).(perms.Permissions)
	if !ok {
		c.Logger().Error("permissions not extracted")
		return fmt.Errorf("permissions not extracted")
	}
	return c.JSON(http.StatusOK, makeUser(user, permissions))
}

type updateUserForm struct {
	FirstName  *string `json:"first_name"`
	LastName   *string `json:"last_name"`
	MiddleName *string `json:"middle_name"`
}

func (f updateUserForm) Update(c echo.Context, user *models.User) error {
	errors := errorFields{}
	if f.FirstName != nil && len(*f.FirstName) > 0 {
		validateFirstName(c, errors, *f.FirstName)
	}
	if f.LastName != nil && len(*f.LastName) > 0 {
		validateLastName(c, errors, *f.LastName)
	}
	if f.MiddleName != nil && len(*f.MiddleName) > 0 {
		validateMiddleName(c, errors, *f.MiddleName)
	}
	if len(errors) > 0 {
		return errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	if f.FirstName != nil {
		user.FirstName = NString(*f.FirstName)
	}
	if f.LastName != nil {
		user.LastName = NString(*f.LastName)
	}
	if f.MiddleName != nil {
		user.MiddleName = NString(*f.MiddleName)
	}
	return nil
}

func (v *View) updateUser(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		c.Logger().Error("user not extracted")
		return fmt.Errorf("user not extracted")
	}
	permissions, ok := c.Get(permissionCtxKey).(perms.Permissions)
	if !ok {
		return fmt.Errorf("permissions not extracted")
	}
	var form updateUserForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var missingPermissions []string
	if form.FirstName != nil {
		if !permissions.HasPermission(perms.UpdateUserFirstNameRole) {
			missingPermissions = append(missingPermissions, perms.UpdateUserFirstNameRole)
		}
	}
	if form.LastName != nil {
		if !permissions.HasPermission(perms.UpdateUserLastNameRole) {
			missingPermissions = append(missingPermissions, perms.UpdateUserLastNameRole)
		}
	}
	if form.MiddleName != nil {
		if !permissions.HasPermission(perms.UpdateUserMiddleNameRole) {
			missingPermissions = append(missingPermissions, perms.UpdateUserMiddleNameRole)
		}
	}
	if len(missingPermissions) > 0 {
		return errorResponse{
			Code:               http.StatusForbidden,
			Message:            localize(c, "Account missing permissions."),
			MissingPermissions: missingPermissions,
		}
	}
	if err := form.Update(c, &user); err != nil {
		c.Logger().Warn(err)
		return err
	}
	if err := v.core.Users.Update(getContext(c), user); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, makeUser(user, permissions))
}

type updateUserStatusForm struct {
	CurrentPassword string `json:"current_password"`
	Status          string `json:"status"`
}

func (f updateUserStatusForm) Update(c echo.Context, user *models.User) error {
	errors := errorFields{}
	switch f.Status {
	case models.PendingUser.String():
		user.Status = models.PendingUser
	case models.ActiveUser.String():
		user.Status = models.ActiveUser
	case models.BlockedUser.String():
		user.Status = models.BlockedUser
	default:
		errors["status"] = errorField{
			Message: localize(c, "Invalid user status."),
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

func (v *View) updateUserStatus(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		c.Logger().Error("user not extracted")
		return fmt.Errorf("user not extracted")
	}
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		c.Logger().Error("auth not extracted")
		return fmt.Errorf("auth not extracted")
	}
	permissions, ok := c.Get(permissionCtxKey).(perms.Permissions)
	if !ok {
		return fmt.Errorf("permissions not extracted")
	}
	var form updateUserStatusForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if err := form.Update(c, &user); err != nil {
		c.Logger().Warn(err)
		return err
	}
	authUser := accountCtx.User
	if authUser == nil ||
		len(form.CurrentPassword) == 0 ||
		!v.core.Users.CheckPassword(*authUser, form.CurrentPassword) {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Invalid password."),
		}
	}
	if err := v.core.Users.Update(getContext(c), user); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, makeUser(user, permissions))
}

type updatePasswordForm struct {
	CurrentPassword string `json:"current_password"`
	Password        string `json:"password"`
}

func (f updatePasswordForm) Update(
	c echo.Context, user *models.User, users *models.UserStore,
) error {
	errors := errorFields{}
	validatePassword(c, errors, f.Password)
	if len(errors) > 0 {
		return errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	if err := users.SetPassword(user, f.Password); err != nil {
		return errorResponse{
			Code:    http.StatusInternalServerError,
			Message: localize(c, "Can not set password."),
		}
	}
	return nil
}

func (v *View) updateUserPassword(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		c.Logger().Error("user not extracted")
		return fmt.Errorf("user not extracted")
	}
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		c.Logger().Error("auth not extracted")
		return fmt.Errorf("auth not extracted")
	}
	permissions, ok := c.Get(permissionCtxKey).(perms.Permissions)
	if !ok {
		c.Logger().Error("permissions not extracted")
		return fmt.Errorf("permissions not extracted")
	}
	var form updatePasswordForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	authUser := accountCtx.User
	if authUser == nil ||
		len(form.CurrentPassword) == 0 ||
		!v.core.Users.CheckPassword(*authUser, form.CurrentPassword) {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Invalid password."),
		}
	}
	if authUser.ID == user.ID && form.CurrentPassword == form.Password {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Old and new passwords are the same."),
		}
	}
	if err := form.Update(c, &user, v.core.Users); err != nil {
		return err
	}
	if err := v.core.Users.Update(getContext(c), user); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, makeUser(user, permissions))
}

type updateEmailForm struct {
	CurrentPassword string `json:"current_password"`
	Email           string `json:"email"`
}

func (f updateEmailForm) Update(c echo.Context, user *models.User) error {
	errors := errorFields{}
	validateEmail(c, errors, f.Email)
	if len(errors) > 0 {
		return errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	if f.Email == string(user.Email) {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Form has invalid fields."),
		}
	}
	return nil
}

const emailTokensLimit = 3
const passwordTokensLimit = 3

func (v *View) updateUserEmail(c echo.Context) error {
	now := getNow(c)
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		c.Logger().Error("user not extracted")
		return fmt.Errorf("user not extracted")
	}
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		c.Logger().Error("auth not extracted")
		return fmt.Errorf("auth not extracted")
	}
	permissions, ok := c.Get(permissionCtxKey).(perms.Permissions)
	if !ok {
		c.Logger().Error("permissions not extracted")
		return fmt.Errorf("permissions not extracted")
	}
	var form updateEmailForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	authUser := accountCtx.User
	if authUser == nil ||
		len(form.CurrentPassword) == 0 ||
		!v.core.Users.CheckPassword(*authUser, form.CurrentPassword) {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Invalid password."),
		}
	}
	if err := form.Update(c, &user); err != nil {
		c.Logger().Warn(err)
		return err
	}
	cfg := v.core.Config.SMTP
	if cfg == nil {
		user.Email = models.NString(form.Email)
		if err := v.core.Users.Update(getContext(c), user); err != nil {
			c.Logger().Error(err)
			return err
		}
		return c.JSON(http.StatusOK, makeUser(user, permissions))
	}
	if count, err := v.core.Tokens.GetCountTokens(
		getContext(c), user.ID, models.ConfirmEmailToken, emailTokensLimit,
	); err != nil {
		c.Logger().Warn(err)
		return err
	} else if count >= emailTokensLimit {
		return errorResponse{
			Code:    http.StatusTooManyRequests,
			Message: localize(c, "Too many requests."),
		}
	}
	expires := now.Add(3 * time.Hour)
	token := models.Token{
		AccountID:  user.ID,
		CreateTime: now.Unix(),
		ExpireTime: expires.Unix(),
	}
	if err := token.SetConfig(models.ConfirmEmailTokenConfig{
		Email: form.Email,
	}); err != nil {
		return err
	}
	if err := token.GenerateSecret(); err != nil {
		return err
	}
	if err := v.core.Tokens.Create(getContext(c), &token); err != nil {
		return err
	}
	to := mail.Address{Address: form.Email}
	values := v.getConfirmEmailValues(c, user, token)
	if err := v.sendConfirmEmailMail(c, cfg, to, values); err != nil {
		c.Logger().Error(err)
		if err := v.core.Tokens.Delete(getContext(c), token.ID); err != nil {
			c.Logger().Error("Cannot remove invalid token", logs.Any("token_id", token.ID), err)
		}
		return err
	}
	userResp := makeUser(user, permissions)
	userResp.UnconfirmedEmail = form.Email
	return c.JSON(http.StatusOK, userResp)
}

func (v *View) resendUserEmail(c echo.Context) error {
	now := getNow(c)
	cfg := v.core.Config.SMTP
	if cfg == nil {
		return c.NoContent(http.StatusNotImplemented)
	}
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		c.Logger().Error("user not extracted")
		return fmt.Errorf("user not extracted")
	}
	permissions, ok := c.Get(permissionCtxKey).(perms.Permissions)
	if !ok {
		c.Logger().Error("permissions not extracted")
		return fmt.Errorf("permissions not extracted")
	}
	if user.Status != models.PendingUser {
		return c.NoContent(http.StatusForbidden)
	}
	errors := errorFields{}
	validateEmail(c, errors, string(user.Email))
	if len(errors) > 0 {
		return errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	if count, err := v.core.Tokens.GetCountTokens(
		getContext(c), user.ID, models.ConfirmEmailToken, emailTokensLimit,
	); err != nil {
		c.Logger().Warn(err)
		return err
	} else if count >= emailTokensLimit {
		return errorResponse{
			Code:    http.StatusTooManyRequests,
			Message: localize(c, "Too many requests."),
		}
	}
	expires := now.Add(3 * time.Hour)
	token := models.Token{
		AccountID:  user.ID,
		CreateTime: now.Unix(),
		ExpireTime: expires.Unix(),
	}
	if err := token.SetConfig(models.ConfirmEmailTokenConfig{
		Email: string(user.Email),
	}); err != nil {
		return err
	}
	if err := token.GenerateSecret(); err != nil {
		return err
	}
	if err := v.core.Tokens.Create(getContext(c), &token); err != nil {
		return err
	}
	to := mail.Address{Address: string(user.Email)}
	values := v.getConfirmEmailValues(c, user, token)
	if err := v.sendConfirmEmailMail(c, cfg, to, values); err != nil {
		c.Logger().Error(err)
		if err := v.core.Tokens.Delete(getContext(c), token.ID); err != nil {
			c.Logger().Error("Cannot remove invalid token", logs.Any("token_id", token.ID), err)
		}
		return err
	}
	return c.JSON(http.StatusOK, makeUser(user, permissions))
}

func (v *View) getConfirmEmailValues(c echo.Context, user models.User, token models.Token) map[string]any {
	return map[string]any{
		"site_url": v.core.Config.Server.SiteURL,
		"login":    user.Login,
		"id":       token.ID,
		"secret":   token.Secret,
	}
}

func (v *View) getResetPasswordValues(c echo.Context, user models.User, token models.Token) map[string]any {
	return map[string]any{
		"site_url": v.core.Config.Server.SiteURL,
		"login":    user.Login,
		"id":       token.ID,
		"secret":   token.Secret,
	}
}

func (v *View) observeUserSessions(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		c.Logger().Error("user not extracted")
		return fmt.Errorf("user not extracted")
	}
	sessions, err := v.core.Sessions.FindByAccount(user.ID)
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	defer func() { _ = sessions.Close() }()
	var resp Sessions
	for sessions.Next() {
		session := sessions.Row()
		resp.Sessions = append(resp.Sessions, Session{
			ID:         session.ID,
			ExpireTime: session.ExpireTime,
			CreateTime: session.CreateTime,
		})
	}
	if err := sessions.Err(); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// status returns current authorization status.
func (v *View) status(c echo.Context) error {
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		c.Logger().Error("auth not extracted")
		return fmt.Errorf("auth not extracted")
	}
	status := Status{}
	if session, ok := c.Get(authSessionKey).(models.Session); ok {
		status.Session = &Session{
			ID:         session.ID,
			CreateTime: session.CreateTime,
			ExpireTime: session.ExpireTime,
		}
	}
	if user := accountCtx.User; user != nil {
		status.User = &User{
			ID:     user.ID,
			Login:  user.Login,
			Status: user.Status.String(),
		}
	}
	if user := accountCtx.ScopeUser; user != nil {
		status.ScopeUser = &ScopeUser{ID: user.ID, Login: user.Login}
	}
	for permission := range accountCtx.Permissions {
		status.Permissions = append(status.Permissions, permission)
	}
	if l := getLocale(c); l != nil {
		status.Locale = l.Name()
	}
	sort.Strings(status.Permissions)
	return c.JSON(http.StatusOK, status)
}

// loginAccount creates a new session for account.
func (v *View) loginAccount(c echo.Context) error {
	now := getNow(c)
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if !ok {
		c.Logger().Error("auth not extracted")
		return fmt.Errorf("auth not extracted")
	}
	if accountCtx.Account.Kind == models.ScopeAccountKind {
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "User not found."),
		}
	}
	expires := now.AddDate(0, 0, 90)
	session := models.Session{
		AccountID:  accountCtx.Account.ID,
		CreateTime: now.Unix(),
		ExpireTime: expires.Unix(),
		RealIP:     c.RealIP(),
		UserAgent:  c.Request().UserAgent(),
	}
	if err := session.GenerateSecret(); err != nil {
		c.Logger().Error(err)
		return err
	}
	if err := v.core.Sessions.Create(getContext(c), &session); err != nil {
		c.Logger().Error(err)
		return err
	}
	cookie := session.Cookie()
	cookie.Name = sessionCookie
	if v.core.Config.Security != nil {
		cookie.Path = v.core.Config.Security.CookiePath
	}
	c.SetCookie(&cookie)
	return c.JSON(http.StatusCreated, Session{
		ID:         session.ID,
		CreateTime: session.CreateTime,
		ExpireTime: session.ExpireTime,
	})
}

// logoutAccount removes current session.
func (v *View) logoutAccount(c echo.Context) error {
	session := c.Get(authSessionKey).(models.Session)
	if err := v.core.Sessions.Delete(getContext(c), session.ID); err != nil && err != sql.ErrNoRows {
		return err
	}
	cookie := http.Cookie{
		Name: sessionCookie,
	}
	if v.core.Config.Security != nil {
		cookie.Path = v.core.Config.Security.CookiePath
	}
	c.SetCookie(&cookie)
	return c.NoContent(http.StatusOK)
}

var loginRegexp = regexp.MustCompile(
	`^[a-zA-Z]([a-zA-Z0-9_\\-])*[a-zA-Z0-9]$`,
)

var emailRegexp = regexp.MustCompile(
	"^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$",
)

func validateLogin(c echo.Context, errors errorFields, login string) {
	if len(login) < 3 {
		errors["login"] = errorField{
			Message: localize(c, "Login too short."),
		}
	} else if len(login) > 20 {
		errors["login"] = errorField{
			Message: localize(c, "Login too long."),
		}
	} else if !loginRegexp.MatchString(login) {
		errors["login"] = errorField{
			Message: localize(c, "Login has invalid format."),
		}
	}
}

func validateEmail(c echo.Context, errors errorFields, email string) {
	if len(email) < 3 {
		errors["email"] = errorField{
			Message: localize(c, "Email too short."),
		}
	} else if len(email) > 254 {
		errors["email"] = errorField{
			Message: localize(c, "Email too long."),
		}
	} else if !emailRegexp.MatchString(email) {
		errors["email"] = errorField{
			Message: localize(c, "Email has invalid format."),
		}
	} else {
		parts := strings.SplitN(email, "@", 2)
		if len(parts) != 2 {
			errors["email"] = errorField{
				Message: localize(c, "Email has invalid format."),
			}
		} else {
			mx, err := net.LookupMX(parts[1])
			if err != nil || len(mx) == 0 {
				errors["email"] = errorField{
					Message: localize(
						c, "Mail server \"{host}\" not responding.",
						replaceField("host", parts[1]),
					),
				}
			}
		}
	}
}

func validatePassword(c echo.Context, errors errorFields, password string) {
	if len(password) < 6 {
		errors["password"] = errorField{
			Message: localize(c, "Password too short."),
		}
	} else if len(password) > 32 {
		errors["password"] = errorField{
			Message: localize(c, "Password too long."),
		}
	}
}

func validateFirstName(c echo.Context, errors errorFields, firstName string) {
	if len(firstName) < 2 {
		errors["first_name"] = errorField{
			Message: localize(c, "First name too short."),
		}
	} else if len(firstName) > 32 {
		errors["first_name"] = errorField{
			Message: localize(c, "First name too long."),
		}
	}
}

func validateLastName(c echo.Context, errors errorFields, lastName string) {
	if len(lastName) < 2 {
		errors["last_name"] = errorField{
			Message: localize(c, "Last name too short."),
		}
	} else if len(lastName) > 32 {
		errors["last_name"] = errorField{
			Message: localize(c, "Last name too long."),
		}
	}
}

func validateMiddleName(c echo.Context, errors errorFields, middleName string) {
	if len(middleName) < 2 {
		errors["middle_name"] = errorField{
			Message: localize(c, "Middle name too short."),
		}
	} else if len(middleName) > 32 {
		errors["middle_name"] = errorField{
			Message: localize(c, "Middle name too long."),
		}
	}
}

// RegisterUserForm represents form for register new user.
type RegisterUserForm struct {
	Login      string `json:"login" form:"login"`
	Email      string `json:"email" form:"email"`
	Password   string `json:"password" form:"password"`
	FirstName  string `json:"first_name" form:"first_name"`
	LastName   string `json:"last_name" form:"last_name"`
	MiddleName string `json:"middle_name" form:"middle_name"`
}

func (f RegisterUserForm) Update(
	c echo.Context, user *models.User, store *models.UserStore,
) error {
	errors := errorFields{}
	validateLogin(c, errors, f.Login)
	validateEmail(c, errors, f.Email)
	validatePassword(c, errors, f.Password)
	if len(f.FirstName) > 0 {
		validateFirstName(c, errors, f.FirstName)
	}
	if len(f.LastName) > 0 {
		validateLastName(c, errors, f.LastName)
	}
	if len(f.MiddleName) > 0 {
		validateMiddleName(c, errors, f.MiddleName)
	}
	if len(errors) > 0 {
		return errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	if _, err := store.GetByLogin(getContext(c), f.Login); err != sql.ErrNoRows {
		if err != nil {
			return errorResponse{
				Code:    http.StatusInternalServerError,
				Message: localize(c, "Unknown error."),
			}
		}
		return errorResponse{
			Code: http.StatusBadRequest,
			Message: localize(
				c, "User with login \"{login}\" already exists.",
				replaceField("login", f.Login),
			),
		}
	}
	user.Login = f.Login
	if err := store.SetPassword(user, f.Password); err != nil {
		return errorResponse{
			Code:    http.StatusInternalServerError,
			Message: localize(c, "Can not set password."),
		}
	}
	user.Email = NString(f.Email)
	user.FirstName = NString(f.FirstName)
	user.LastName = NString(f.LastName)
	user.MiddleName = NString(f.MiddleName)
	return nil
}

// registerUser registers user.
func (v *View) registerUser(c echo.Context) error {
	var form RegisterUserForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	now := getNow(c)
	user := models.User{}
	if err := form.Update(c, &user, v.core.Users); err != nil {
		return err
	}
	user.Status = models.PendingUser
	if v.core.Config.SMTP == nil {
		user.Status = models.ActiveUser
	}
	expires := now.AddDate(0, 0, 1)
	token := models.Token{
		CreateTime: now.Unix(),
		ExpireTime: expires.Unix(),
	}
	if err := token.SetConfig(models.ConfirmEmailTokenConfig{
		Email: string(user.Email),
	}); err != nil {
		return err
	}
	if err := token.GenerateSecret(); err != nil {
		return err
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		account := models.Account{Kind: user.AccountKind()}
		if err := v.core.Accounts.Create(ctx, &account); err != nil {
			return err
		}
		user.ID = account.ID
		if err := v.core.Users.Create(ctx, &user); err != nil {
			return err
		}
		if v.core.Config.SMTP != nil {
			token.AccountID = account.ID
			if err := v.core.Tokens.Create(ctx, &token); err != nil {
				return err
			}
		}
		return nil
	}, sqlRepeatableRead); err != nil {
		c.Logger().Error(err)
		return err
	}
	if cfg := v.core.Config.SMTP; cfg != nil {
		to := mail.Address{Address: string(user.Email)}
		values := v.getConfirmEmailValues(c, user, token)
		if err := v.sendConfirmEmailMail(c, cfg, to, values); err != nil {
			c.Logger().Error(err)
			return err
		}
	}
	return c.JSON(http.StatusCreated, User{
		ID:         user.ID,
		Login:      user.Login,
		Email:      form.Email,
		FirstName:  form.FirstName,
		LastName:   form.LastName,
		MiddleName: form.MiddleName,
	})
}

type resetPasswordForm struct {
	Login string `json:"login"`
}

// resetUserPassword resets user password.
func (v *View) resetUserPassword(c echo.Context) error {
	now := getNow(c)
	cfg := v.core.Config.SMTP
	if cfg == nil {
		return c.NoContent(http.StatusNotImplemented)
	}
	var form resetPasswordForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	ctx := getContext(c)
	user, err := v.core.Users.GetByLogin(ctx, form.Login)
	if err != nil {
		return err
	}
	if count, err := v.core.Tokens.GetCountTokens(
		ctx, user.ID, models.ResetPasswordToken, passwordTokensLimit,
	); err != nil {
		c.Logger().Warn(err)
		return err
	} else if count >= passwordTokensLimit {
		return errorResponse{
			Code:    http.StatusTooManyRequests,
			Message: localize(c, "Too many requests."),
		}
	}
	expires := now.Add(30 * time.Minute)
	token := models.Token{
		AccountID:  user.ID,
		CreateTime: now.Unix(),
		ExpireTime: expires.Unix(),
	}
	if err := token.SetConfig(models.ResetPasswordTokenConfig{}); err != nil {
		return err
	}
	if err := token.GenerateSecret(); err != nil {
		return err
	}
	if err := v.core.Tokens.Create(ctx, &token); err != nil {
		return err
	}
	to := mail.Address{Address: string(user.Email)}
	values := v.getResetPasswordValues(c, user, token)
	if err := v.sendResetPasswordMail(c, cfg, to, values); err != nil {
		c.Logger().Error(err)
		if err := v.core.Tokens.Delete(ctx, token.ID); err != nil {
			c.Logger().Error("Cannot remove invalid token", logs.Any("token_id", token.ID), err)
		}
		return err
	}
	userResp := User{Login: user.Login}
	return c.JSON(http.StatusOK, userResp)
}

func (v *View) sendConfirmEmailMail(c echo.Context, cfg *config.SMTP, to mail.Address, values map[string]any) error {
	return v.sendMail(
		c,
		cfg,
		to,
		"confirm_email",
		values,
		"Email confirmation on Solve",
		"Confirmation link: {{.site_url}}/confirm-email?id={{.id}}&secret={{.secret}}",
	)
}

func (v *View) sendResetPasswordMail(c echo.Context, cfg *config.SMTP, to mail.Address, values map[string]any) error {
	return v.sendMail(
		c,
		cfg,
		to,
		"reset_password",
		values,
		"Reset password on Solve",
		"Reset link: {{.site_url}}/reset-password?id={{.id}}&secret={{.secret}}",
	)
}

func (v *View) sendMail(
	c echo.Context,
	cfg *config.SMTP,
	to mail.Address,
	key string,
	values map[string]any,
	defaultSubject string,
	defaultBody string,
) error {
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return err
	}
	if err := client.Auth(smtp.PlainAuth("", cfg.Email, cfg.Password, cfg.Host)); err != nil {
		return err
	}
	if err := client.Mail(cfg.Email); err != nil {
		return err
	}
	if err := client.Rcpt(to.Address); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	defer writer.Close()
	from := mail.Address{Name: cfg.Name, Address: cfg.Email}
	locale := getLocale(c)
	subject, err := renderTemplate(locale.LocalizeKey(key+".subject", defaultSubject), values)
	if err != nil {
		return err
	}
	body, err := renderTemplate(locale.LocalizeKey(key+".body", defaultBody), values)
	if err != nil {
		return err
	}
	if _, err := writer.Write([]byte(fmt.Sprintf("From: %s\r\n", from.String()))); err != nil {
		return err
	}
	if _, err := writer.Write([]byte(fmt.Sprintf("To: %s\r\n", to.String()))); err != nil {
		return err
	}
	if _, err := writer.Write([]byte(fmt.Sprintf("Subject: %s\r\n", subject))); err != nil {
		return err
	}
	if _, err := writer.Write([]byte("\r\n")); err != nil {
		return err
	}
	if _, err := writer.Write([]byte(body)); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func renderTemplate(text string, values map[string]any) (string, error) {
	tmpl, err := template.New("").Parse(text)
	if err != nil {
		return "", err
	}
	b := strings.Builder{}
	if err := tmpl.Execute(&b, values); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (v *View) extractUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		login := c.Param("user")
		accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
		if !ok {
			return fmt.Errorf("auth not extracted")
		}
		if err := syncStore(c, v.core.Users); err != nil {
			return err
		}
		id, err := strconv.ParseInt(login, 10, 64)
		if err != nil {
			user, err := v.core.Users.GetByLogin(getContext(c), login)
			if err != nil {
				if err == sql.ErrNoRows {
					return errorResponse{
						Code: http.StatusNotFound,
						Message: localize(
							c, "User \"{login}\" does not exists.",
							replaceField("login", login),
						),
					}
				}
				return err
			}
			c.Set(userKey, user)
			c.Set(permissionCtxKey, v.getUserPermissions(accountCtx, user))
			return next(c)
		}
		user, err := v.core.Users.Get(getContext(c), id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code: http.StatusNotFound,
					Message: localize(
						c, "User {id} does not exists.",
						replaceField("id", id),
					),
				}
			}
			return err
		}
		c.Set(userKey, user)
		c.Set(permissionCtxKey, v.getUserPermissions(accountCtx, user))
		return next(c)
	}
}

func (v *View) getUserPermissions(
	ctx *managers.AccountContext, user models.User,
) perms.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if authUser := ctx.User; authUser != nil && authUser.ID == user.ID {
		permissions.AddPermission(
			perms.ObserveUserEmailRole,
			perms.ObserveUserMiddleNameRole,
			perms.ObserveUserStatusRole,
			perms.ObserveUserSessionsRole,
			perms.UpdateUserRole,
			perms.UpdateUserPasswordRole,
			perms.UpdateUserEmailRole,
			perms.UpdateUserFirstNameRole,
			perms.UpdateUserLastNameRole,
			perms.UpdateUserMiddleNameRole,
		)
	}
	permissions.AddPermission(
		perms.ObserveUserFirstNameRole,
		perms.ObserveUserLastNameRole,
	)
	return permissions
}

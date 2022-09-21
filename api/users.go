package api

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

// User represents user.
type User struct {
	// ID contains user ID.
	ID int64 `json:"id"`
	// Login contains user login.
	Login string `json:"login"`
	// Email contains user email.
	Email string `json:"email,omitempty"`
	// FirstName contains first name.
	FirstName string `json:"first_name,omitempty"`
	// LastName contains last name.
	LastName string `json:"last_name,omitempty"`
	// MiddleName contains middle name.
	MiddleName string `json:"middle_name,omitempty"`
}

// Status represents current authorization status.
type Status struct {
	User        *User    `json:"user,omitempty"`
	Session     *Session `json:"session,omitempty"`
	Permissions []string `json:"permissions"`
}

// registerUserHandlers registers handlers for user management.
func (v *View) registerUserHandlers(g *echo.Group) {
	if v.core.Users == nil {
		return
	}
	g.GET(
		"/v0/users/:user", v.observeUser,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractUser,
		v.requirePermission(models.ObserveUserRole),
	)
	g.PATCH(
		"/v0/users/:user", v.updateUser,
		v.extractAuth(v.sessionAuth), v.extractUser,
		v.requirePermission(models.UpdateUserRole),
	)
	g.GET(
		"/v0/users/:user/sessions", v.observeUserSessions,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractUser,
		v.requirePermission(models.ObserveUserSessionsRole),
	)
	g.POST(
		"/v0/users/:user/password", v.updateUserPassword,
		v.extractAuth(v.sessionAuth), v.extractUser,
		v.requirePermission(models.UpdateUserPasswordRole),
	)
	g.GET(
		"/v0/status", v.status,
		v.extractAuth(v.sessionAuth, v.userAuth, v.guestAuth),
		v.requirePermission(models.StatusRole),
	)
	g.POST(
		"/v0/login", v.loginAccount,
		v.extractAuth(v.userAuth),
		v.requirePermission(models.LoginRole),
	)
	g.POST(
		"/v0/logout", v.logoutAccount,
		v.extractAuth(v.sessionAuth),
		v.requirePermission(models.LogoutRole),
	)
	g.POST(
		"/v0/register", v.registerUser,
		v.extractAuth(v.sessionAuth, v.guestAuth),
		v.requirePermission(models.RegisterRole),
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

func makeUser(user models.User, permissions managers.Permissions) User {
	assign := func(field *string, value, permission string) {
		if permissions.HasPermission(permission) {
			*field = value
		}
	}
	resp := User{ID: user.ID, Login: user.Login}
	assign(&resp.Email, string(user.Email), models.ObserveUserEmailRole)
	assign(&resp.FirstName, string(user.FirstName), models.ObserveUserFirstNameRole)
	assign(&resp.LastName, string(user.LastName), models.ObserveUserLastNameRole)
	assign(&resp.MiddleName, string(user.MiddleName), models.ObserveUserMiddleNameRole)
	return resp
}

func (v *View) observeUser(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		c.Logger().Error("user not extracted")
		return fmt.Errorf("user not extracted")
	}
	permissions, ok := c.Get(permissionCtxKey).(managers.Permissions)
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

func (f updateUserForm) Update(user *models.User) *errorResponse {
	errors := errorFields{}
	if f.FirstName != nil && len(*f.FirstName) > 0 {
		validateFirstName(errors, *f.FirstName)
	}
	if f.LastName != nil && len(*f.LastName) > 0 {
		validateLastName(errors, *f.LastName)
	}
	if f.MiddleName != nil && len(*f.MiddleName) > 0 {
		validateMiddleName(errors, *f.MiddleName)
	}
	if len(errors) > 0 {
		return &errorResponse{
			Message:       "passed invalid fields to form",
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
	permissions, ok := c.Get(permissionCtxKey).(managers.Permissions)
	if !ok {
		c.Logger().Error("permissions not extracted")
		return fmt.Errorf("permissions not extracted")
	}
	var form updateUserForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var missingPermissions []string
	if form.FirstName != nil {
		if !permissions.HasPermission(models.UpdateUserFirstNameRole) {
			missingPermissions = append(missingPermissions, models.UpdateUserFirstNameRole)
		}
	}
	if form.LastName != nil {
		if !permissions.HasPermission(models.UpdateUserLastNameRole) {
			missingPermissions = append(missingPermissions, models.UpdateUserLastNameRole)
		}
	}
	if form.MiddleName != nil {
		if !permissions.HasPermission(models.UpdateUserMiddleNameRole) {
			missingPermissions = append(missingPermissions, models.UpdateUserMiddleNameRole)
		}
	}
	if len(missingPermissions) > 0 {
		return c.JSON(http.StatusForbidden, errorResponse{
			Message:            "account missing permissions",
			MissingPermissions: missingPermissions,
		})
	}
	if err := form.Update(&user); err != nil {
		c.Logger().Warn(err)
		return c.JSON(http.StatusBadRequest, err)
	}
	if err := v.core.Users.Update(getContext(c), user); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, makeUser(user, permissions))
}

type updatePasswordForm struct {
	OldPassword string `json:"old_password"`
	Password    string `json:"password"`
}

func (f updatePasswordForm) Update(user *models.User, users *models.UserStore) *errorResponse {
	errors := errorFields{}
	validatePassword(errors, f.Password)
	if len(errors) > 0 {
		return &errorResponse{
			Message:       "passed invalid fields to form",
			InvalidFields: errors,
		}
	}
	if f.OldPassword == f.Password {
		return &errorResponse{
			Message: "old and new passwords are the same",
		}
	}
	if err := users.SetPassword(user, f.Password); err != nil {
		return &errorResponse{
			Message: "unable to change old password",
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
	permissions, ok := c.Get(permissionCtxKey).(managers.Permissions)
	if !ok {
		c.Logger().Error("permissions not extracted")
		return fmt.Errorf("permissions not extracted")
	}
	var form updatePasswordForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if authUser := accountCtx.User; authUser != nil && user.ID == authUser.ID {
		if len(form.OldPassword) == 0 {
			return c.JSON(http.StatusBadRequest, errorResponse{
				Message: "old password should not be empty",
			})
		}
		if !v.core.Users.CheckPassword(user, form.OldPassword) {
			return c.JSON(http.StatusBadRequest, errorResponse{
				Message: "entered invalid old password",
			})
		}
	}
	if err := form.Update(&user, v.core.Users); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	if err := v.core.Users.Update(getContext(c), user); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, makeUser(user, permissions))
}

func (v *View) observeUserSessions(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		c.Logger().Error("user not extracted")
		return fmt.Errorf("user not extracted")
	}
	sessions, err := v.core.Sessions.FindByAccount(user.AccountID)
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	var resp Sessions
	for _, session := range sessions {
		resp.Sessions = append(resp.Sessions, Session{
			ID:         session.ID,
			ExpireTime: session.ExpireTime,
			CreateTime: session.CreateTime,
		})
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
		status.User = &User{ID: user.ID, Login: user.Login}
	}
	for permission := range accountCtx.Permissions {
		status.Permissions = append(status.Permissions, permission)
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
	expires := now.AddDate(0, 0, 90)
	session := models.Session{
		AccountID:  accountCtx.Account.ID,
		CreateTime: now.Unix(),
		ExpireTime: expires.Unix(),
		RemoteAddr: c.Request().RemoteAddr,
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
	if err := v.core.Sessions.Delete(getContext(c), session.ID); err != nil {
		c.Logger().Error(err)
		return err
	}
	cookie := http.Cookie{Name: sessionCookie}
	c.SetCookie(&cookie)
	return c.NoContent(http.StatusOK)
}

var loginRegexp = regexp.MustCompile(
	`^[a-zA-Z]([a-zA-Z0-9_\\-])*[a-zA-Z0-9]$`,
)

var emailRegexp = regexp.MustCompile(
	"^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$",
)

func validateLogin(errors errorFields, login string) {
	if len(login) < 3 {
		errors["login"] = errorField{Message: "login too short (<3)"}
	} else if len(login) > 20 {
		errors["login"] = errorField{Message: "login too long (>20)"}
	} else if !loginRegexp.MatchString(login) {
		errors["login"] = errorField{Message: "login has invalid format"}
	}
}

func validateEmail(errors errorFields, email string) {
	if len(email) < 3 {
		errors["email"] = errorField{Message: "email too short (<3)"}
	} else if len(email) > 254 {
		errors["email"] = errorField{Message: "email too long (>254)"}
	} else if !emailRegexp.MatchString(email) {
		errors["email"] = errorField{Message: "email has invalid format"}
	} else {
		parts := strings.SplitN(email, "@", 2)
		if len(parts) != 2 {
			errors["email"] = errorField{Message: "email has invalid format"}
		} else {
			mx, err := net.LookupMX(parts[1])
			if err != nil || len(mx) == 0 {
				errors["email"] = errorField{
					Message: fmt.Sprintf(
						"mailserver %q not found", parts[1],
					),
				}
			}
		}
	}
}

func validatePassword(errors errorFields, password string) {
	if len(password) < 6 {
		errors["password"] = errorField{Message: "password too short (<6)"}
	} else if len(password) > 32 {
		errors["password"] = errorField{Message: "password too long (>32)"}
	}
}

func validateFirstName(errors errorFields, firstName string) {
	if len(firstName) < 2 {
		errors["first_name"] = errorField{Message: "first name too short (<2)"}
	} else if len(firstName) > 32 {
		errors["first_name"] = errorField{Message: "first name too long (>32)"}
	}
}

func validateLastName(errors errorFields, lastName string) {
	if len(lastName) < 2 {
		errors["last_name"] = errorField{Message: "last name too short (<2)"}
	} else if len(lastName) > 32 {
		errors["last_name"] = errorField{Message: "last name too long (>32)"}
	}
}

func validateMiddleName(errors errorFields, middleName string) {
	if len(middleName) < 2 {
		errors["middle_name"] = errorField{Message: "middle name too short (<2)"}
	} else if len(middleName) > 32 {
		errors["middle_name"] = errorField{Message: "middle name too long (>32)"}
	}
}

func (f RegisterUserForm) validate() *errorResponse {
	errors := errorFields{}
	validateLogin(errors, f.Login)
	validateEmail(errors, f.Email)
	validatePassword(errors, f.Password)
	if len(f.FirstName) > 0 {
		validateFirstName(errors, f.FirstName)
	}
	if len(f.LastName) > 0 {
		validateLastName(errors, f.LastName)
	}
	if len(f.MiddleName) > 0 {
		validateMiddleName(errors, f.MiddleName)
	}
	if len(errors) == 0 {
		return nil
	}
	return &errorResponse{
		Message:       "passed invalid fields to form",
		InvalidFields: errors,
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
	user *models.User, store *models.UserStore,
) *errorResponse {
	if err := f.validate(); err != nil {
		return err
	}
	if _, err := store.GetByLogin(f.Login); err != sql.ErrNoRows {
		if err != nil {
			return &errorResponse{Message: "unknown error"}
		}
		return &errorResponse{
			Message: fmt.Sprintf(
				"user with login %q already exists", f.Login,
			),
		}
	}
	user.Login = f.Login
	if err := store.SetPassword(user, f.Password); err != nil {
		return &errorResponse{Message: "can not set password"}
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
	var user models.User
	if err := form.Update(&user, v.core.Users); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	if err := v.core.WrapTx(getContext(c), func(ctx context.Context) error {
		account := models.Account{Kind: user.AccountKind()}
		if err := v.core.Accounts.Create(ctx, &account); err != nil {
			return err
		}
		user.AccountID = account.ID
		return v.core.Users.Create(ctx, &user)
	}, sqlRepeatableRead); err != nil {
		c.Logger().Error(err)
		return err
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

func (v *View) extractUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		login := c.Param("user")
		accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
		if !ok {
			return fmt.Errorf("auth not extracted")
		}
		id, err := strconv.ParseInt(login, 10, 64)
		if err != nil {
			user, err := v.core.Users.GetByLogin(login)
			if err != nil {
				if err == sql.ErrNoRows {
					return errorResponse{
						Code:    http.StatusNotFound,
						Message: fmt.Sprintf("user %q not found", login),
					}
				}
				return err
			}
			c.Set(userKey, user)
			c.Set(permissionCtxKey, v.getUserPermissions(accountCtx, user))
			return next(c)
		}
		user, err := v.core.Users.Get(id)
		if err != nil {
			if err == sql.ErrNoRows {
				return errorResponse{
					Code:    http.StatusNotFound,
					Message: fmt.Sprintf("user %d not found", id),
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
) managers.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if authUser := ctx.User; authUser != nil && authUser.ID == user.ID {
		permissions[models.ObserveUserEmailRole] = struct{}{}
		permissions[models.ObserveUserMiddleNameRole] = struct{}{}
		permissions[models.ObserveUserSessionsRole] = struct{}{}
		permissions[models.UpdateUserRole] = struct{}{}
		permissions[models.UpdateUserPasswordRole] = struct{}{}
		permissions[models.UpdateUserEmailRole] = struct{}{}
		permissions[models.UpdateUserFirstNameRole] = struct{}{}
		permissions[models.UpdateUserLastNameRole] = struct{}{}
		permissions[models.UpdateUserMiddleNameRole] = struct{}{}
	}
	permissions[models.ObserveUserFirstNameRole] = struct{}{}
	permissions[models.ObserveUserLastNameRole] = struct{}{}
	return permissions
}

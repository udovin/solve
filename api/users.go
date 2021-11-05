package api

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/core"
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

// Session represents session.
type Session struct {
	// ID contains session ID.
	ID int64 `json:"id"`
	// CreateTime contains session create time.
	CreateTime int64 `json:"create_time,omitempty"`
	// ExpireTime contains session expire time.
	ExpireTime int64 `json:"expire_time,omitempty"`
}

// Status represents current authorization status.
type Status struct {
	User    *User    `json:"user,omitempty"`
	Session *Session `json:"session,omitempty"`
	Roles   []string `json:"roles"`
}

// registerUserHandlers registers handlers for user management.
func (v *View) registerUserHandlers(g *echo.Group) {
	g.GET(
		"/users/:user", v.observeUser,
		v.sessionAuth, v.extractUser, v.extractUserRoles,
		v.requireAuthRole(models.ObserveUserRole),
	)
	g.PATCH(
		"/users/:user", v.updateUser,
		v.sessionAuth, v.extractUser, v.extractUserRoles,
		v.requireAuthRole(models.UpdateUserRole),
	)
	g.GET(
		"/users/:user/sessions", v.observeUserSessions,
		v.sessionAuth, v.extractUser, v.extractUserRoles,
		v.requireAuthRole(models.ObserveUserSessionsRole),
	)
	g.POST(
		"/users/:user/password", v.updateUserPassword,
		v.sessionAuth, v.extractUser, v.extractUserRoles,
		v.requireAuthRole(models.UpdateUserPasswordRole),
	)
	g.GET(
		"/status", v.status,
		v.sessionAuth,
		v.requireAuthRole(models.StatusRole),
	)
	g.POST(
		"/login", v.loginAccount,
		v.userAuth, v.requireAuth,
		v.requireAuthRole(models.LoginRole),
	)
	g.POST(
		"/logout", v.logoutAccount,
		v.sessionAuth, v.requireAuth,
		v.requireAuthRole(models.LogoutRole),
	)
	g.POST(
		"/register", v.registerUser,
		v.requireAuthRole(models.RegisterRole),
	)
}

func (v *View) registerSocketUserHandlers(g *echo.Group) {
	g.GET(
		"/users/:user", v.observeUser, v.extractUser,
		v.extractAuthRoles,
	)
}

func makeUser(c echo.Context, user models.User, roles core.RoleSet, core *core.Core) User {
	assign := func(field *string, value, role string) {
		if ok, err := core.HasRole(roles, role); ok {
			*field = value
		} else if err != nil {
			c.Logger().Error(err)
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
	roles, ok := c.Get(authRolesKey).(core.RoleSet)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	return c.JSON(http.StatusOK, makeUser(c, user, roles, v.core))
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
		user.FirstName = models.NString(*f.FirstName)
	}
	if f.LastName != nil {
		user.LastName = models.NString(*f.LastName)
	}
	if f.MiddleName != nil {
		user.MiddleName = models.NString(*f.MiddleName)
	}
	return nil
}

func (v *View) updateUser(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		c.Logger().Error("user not extracted")
		return fmt.Errorf("user not extracted")
	}
	roles, ok := c.Get(authRolesKey).(core.RoleSet)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	var form updateUserForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var missingRoles []string
	if form.FirstName != nil {
		ok, err := v.core.HasRole(roles, models.UpdateUserFirstNameRole)
		if err != nil {
			c.Logger().Error(err)
			return err
		}
		if !ok {
			missingRoles = append(missingRoles, models.UpdateUserFirstNameRole)
		}
	}
	if form.LastName != nil {
		ok, err := v.core.HasRole(roles, models.UpdateUserLastNameRole)
		if err != nil {
			c.Logger().Error(err)
			return err
		}
		if !ok {
			missingRoles = append(missingRoles, models.UpdateUserLastNameRole)
		}
	}
	if form.MiddleName != nil {
		ok, err := v.core.HasRole(roles, models.UpdateUserMiddleNameRole)
		if err != nil {
			c.Logger().Error(err)
			return err
		}
		if !ok {
			missingRoles = append(missingRoles, models.UpdateUserMiddleNameRole)
		}
	}
	if len(missingRoles) > 0 {
		return c.JSON(http.StatusForbidden, errorResponse{
			Message:      "account missing roles",
			MissingRoles: missingRoles,
		})
	}
	if err := form.Update(&user); err != nil {
		c.Logger().Warn(err)
		return c.JSON(http.StatusBadRequest, err)
	}
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		return v.core.Users.UpdateTx(tx, user)
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, makeUser(c, user, roles, v.core))
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
	roles, ok := c.Get(authRolesKey).(core.RoleSet)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	var form updatePasswordForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if authUser, ok := c.Get(authUserKey).(models.User); ok && user.ID == authUser.ID {
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
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		return v.core.Users.UpdateTx(tx, user)
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	return c.JSON(http.StatusOK, makeUser(c, user, roles, v.core))
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
	var resp []Session
	for _, session := range sessions {
		resp = append(resp, Session{
			ID:         session.ID,
			ExpireTime: session.ExpireTime,
			CreateTime: session.CreateTime,
		})
	}
	return c.JSON(http.StatusOK, resp)
}

// status returns current authorization status.
func (v *View) status(c echo.Context) error {
	status := Status{}
	if session, ok := c.Get(authSessionKey).(models.Session); ok {
		status.Session = &Session{
			ID:         session.ID,
			CreateTime: session.CreateTime,
			ExpireTime: session.ExpireTime,
		}
		if user, ok := c.Get(authUserKey).(models.User); ok {
			status.User = &User{ID: user.ID, Login: user.Login}
		}
	}
	if roles, ok := c.Get(authRolesKey).(core.RoleSet); ok {
		for id := range roles {
			if role, err := v.core.Roles.Get(id); err == nil {
				if role.IsBuiltIn() {
					status.Roles = append(status.Roles, role.Code)
				}
			}
		}
	}
	return c.JSON(http.StatusOK, status)
}

// loginAccount creates a new session for account.
func (v *View) loginAccount(c echo.Context) error {
	account, ok := c.Get(authAccountKey).(models.Account)
	if !ok {
		c.Logger().Error("account not extracted")
		return fmt.Errorf("account not extracted")
	}
	created := time.Now()
	expires := created.Add(time.Hour * 24 * 90)
	session := models.Session{
		AccountID:  account.ID,
		CreateTime: created.Unix(),
		ExpireTime: expires.Unix(),
	}
	if err := session.GenerateSecret(); err != nil {
		c.Logger().Error(err)
		return err
	}
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		var err error
		session, err = v.core.Sessions.CreateTx(tx, session)
		return err
	}); err != nil {
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
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		return v.core.Sessions.DeleteTx(tx, session.ID)
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
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

func (f registerUserForm) validate() *errorResponse {
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

// registerUserForm represents form for register new user.
type registerUserForm struct {
	Login      string `json:"login"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	MiddleName string `json:"middle_name"`
}

func (f registerUserForm) Update(
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
	user.Email = models.NString(f.Email)
	user.FirstName = models.NString(f.FirstName)
	user.LastName = models.NString(f.LastName)
	user.MiddleName = models.NString(f.MiddleName)
	return nil
}

// registerUser registers user.
func (v *View) registerUser(c echo.Context) error {
	var form registerUserForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var user models.User
	if err := form.Update(&user, v.core.Users); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		account := models.Account{Kind: models.UserAccount}
		if err := v.core.Accounts.CreateTx(tx, &account); err != nil {
			return err
		}
		user.AccountID = account.ID
		return v.core.Users.CreateTx(tx, &user)
	}); err != nil {
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
		id, err := strconv.ParseInt(c.Param("user"), 10, 64)
		if err != nil {
			user, err := v.core.Users.GetByLogin(c.Param("user"))
			if err != nil {
				if err == sql.ErrNoRows {
					resp := errorResponse{Message: "user not found"}
					return c.JSON(http.StatusNotFound, resp)
				}
				c.Logger().Error(err)
				return err
			}
			c.Set(userKey, user)
			return next(c)
		}
		user, err := v.core.Users.Get(id)
		if err != nil {
			if err == sql.ErrNoRows {
				resp := errorResponse{Message: "user not found"}
				return c.JSON(http.StatusNotFound, resp)
			}
			c.Logger().Error(err)
			return err
		}
		c.Set(userKey, user)
		return next(c)
	}
}

func (v *View) extractUserRoles(next echo.HandlerFunc) echo.HandlerFunc {
	nextWrap := func(c echo.Context) error {
		user, ok := c.Get(userKey).(models.User)
		if !ok {
			c.Logger().Error("user not extracted")
			return fmt.Errorf("user not extracted")
		}
		authRoles, ok := c.Get(authRolesKey).(core.RoleSet)
		if !ok {
			c.Logger().Error("roles not extracted")
			return fmt.Errorf("roles not extracted")
		}
		addRole := func(roles core.RoleSet, code string) {
			if err := v.core.AddRole(roles, code); err != nil {
				c.Logger().Error(err)
			}
		}
		authUser, ok := c.Get(authUserKey).(models.User)
		if ok && authUser.ID == user.ID {
			addRole(authRoles, models.ObserveUserEmailRole)
			addRole(authRoles, models.ObserveUserMiddleNameRole)
			addRole(authRoles, models.ObserveUserSessionsRole)
			addRole(authRoles, models.UpdateUserRole)
			addRole(authRoles, models.UpdateUserPasswordRole)
			addRole(authRoles, models.UpdateUserEmailRole)
			addRole(authRoles, models.UpdateUserFirstNameRole)
			addRole(authRoles, models.UpdateUserLastNameRole)
			addRole(authRoles, models.UpdateUserMiddleNameRole)
		}
		addRole(authRoles, models.ObserveUserFirstNameRole)
		addRole(authRoles, models.ObserveUserLastNameRole)
		return next(c)
	}
	return v.extractAuthRoles(nextWrap)
}

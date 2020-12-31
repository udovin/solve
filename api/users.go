package api

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo"

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
	// Secret contains session secret.
	Secret string `json:"secret,omitempty"`
	// CreateTime contains session create time.
	CreateTime int64 `json:"create_time,omitempty"`
	// ExpireTime contains session expire time.
	ExpireTime int64 `json:"expire_time,omitempty"`
}

// AuthStatus represents current authorization status.
type AuthStatus struct {
	User    *User    `json:"user,omitempty"`
	Session *Session `json:"session,omitempty"`
	Roles   []string `json:"roles"`
}

// registerUserHandlers registers handlers for user management.
func (v *View) registerUserHandlers(g *echo.Group) {
	g.GET(
		"/users/:user", v.observeUser,
		v.sessionAuth,
		v.requireAuthRole(models.ObserveUserRole),
		v.extractUser,
		v.extractUserRoles,
	)
	g.GET(
		"/auth-status", v.authStatus,
		v.sessionAuth,
		v.requireAuthRole(models.AuthStatusRole),
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

func (v *View) observeUser(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		c.Logger().Error("user not extracted")
		return fmt.Errorf("user not extracted")
	}
	roles, ok := c.Get(authRolesKey).(core.Roles)
	if !ok {
		c.Logger().Error("roles not extracted")
		return fmt.Errorf("roles not extracted")
	}
	assign := func(field *string, value, role string) {
		if ok, err := v.core.HasRole(roles, role); ok {
			*field = value
		} else if err != nil {
			c.Logger().Error(err)
		}
	}
	resp := User{ID: user.ID, Login: user.Login}
	fields, err := v.core.UserFields.FindByUser(user.ID)
	if err != nil {
		c.Logger().Error(err)
	} else {
		for _, field := range fields {
			switch field.Kind {
			case models.EmailField:
				assign(
					&resp.Email, field.Data,
					models.ObserveUserEmailRole,
				)
			case models.FirstNameField:
				assign(
					&resp.FirstName, field.Data,
					models.ObserveUserFirstNameRole,
				)
			case models.LastNameField:
				assign(
					&resp.LastName, field.Data,
					models.ObserveUserLastNameRole,
				)
			case models.MiddleNameField:
				assign(
					&resp.MiddleName, field.Data,
					models.ObserveUserMiddleNameRole,
				)
			}
		}
	}
	return c.JSON(http.StatusOK, resp)
}

// authStatus returns current authorization status.
func (v *View) authStatus(c echo.Context) error {
	status := AuthStatus{}
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
	if roles, ok := c.Get(authRolesKey).(core.Roles); ok {
		for id := range roles {
			if role, err := v.core.Roles.Get(id); err == nil {
				status.Roles = append(status.Roles, role.Code)
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
		Secret:     session.Secret,
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

// registerUserForm represents form for register new user.
type registerUserForm struct {
	Login      string `json:"login"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	MiddleName string `json:"middle_name"`
}

var loginRegexp = regexp.MustCompile(
	"^[a-zA-Z]([a-zA-Z0-9_\\-])*[a-zA-Z0-9]$",
)

var emailRegexp = regexp.MustCompile(
	"^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$",
)

func (f registerUserForm) Validate() *errorResp {
	errors := errorFields{}
	validateLogin(errors, f.Login)
	validateEmail(errors, f.Email)
	validatePassword(errors, f.Password)
	if len(errors) == 0 {
		return nil
	}
	return &errorResp{
		Message:       "passed invalid fields to form",
		InvalidFields: errors,
	}
}

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

// registerUser registers user.
func (v *View) registerUser(c echo.Context) error {
	var form registerUserForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if resp := form.Validate(); resp != nil {
		return c.JSON(http.StatusBadRequest, resp)
	}
	if _, err := v.core.Users.GetByLogin(form.Login); err != sql.ErrNoRows {
		if err != nil {
			c.Logger().Error(err)
			return err
		}
		resp := errorResp{
			Message: fmt.Sprintf(
				"user with login %q already exists", form.Login,
			),
		}
		return c.JSON(http.StatusForbidden, resp)
	}
	user := models.User{Login: form.Login}
	if err := v.core.Users.SetPassword(&user, form.Password); err != nil {
		c.Logger().Error(err)
		return err
	}
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		account := models.Account{Kind: models.UserAccount}
		account, err := v.core.Accounts.CreateTx(tx, account)
		if err != nil {
			return err
		}
		user.AccountID = account.ID
		user, err = v.core.Users.CreateTx(tx, user)
		if err != nil {
			return err
		}
		return v.registerUserFields(tx, user, form)
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

// registerUserFields creates fields for registered user.
func (v *View) registerUserFields(
	tx *sql.Tx, user models.User, form registerUserForm,
) error {
	email := models.UserField{
		UserID: user.ID,
		Kind:   models.EmailField,
		Data:   form.Email,
	}
	if _, err := v.core.UserFields.CreateTx(tx, email); err != nil {
		return err
	}
	if form.FirstName != "" {
		field := models.UserField{
			UserID: user.ID,
			Kind:   models.FirstNameField,
			Data:   form.FirstName,
		}
		if _, err := v.core.UserFields.CreateTx(tx, field); err != nil {
			return err
		}
	}
	if form.LastName != "" {
		field := models.UserField{
			UserID: user.ID,
			Kind:   models.LastNameField,
			Data:   form.LastName,
		}
		if _, err := v.core.UserFields.CreateTx(tx, field); err != nil {
			return err
		}
	}
	if form.MiddleName != "" {
		field := models.UserField{
			UserID: user.ID,
			Kind:   models.MiddleNameField,
			Data:   form.MiddleName,
		}
		if _, err := v.core.UserFields.CreateTx(tx, field); err != nil {
			return err
		}
	}
	return nil
}

func (v *View) extractUserRoles(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := c.Get(userKey).(models.User)
		if !ok {
			c.Logger().Error("user not extracted")
			return fmt.Errorf("user not extracted")
		}
		roles, ok := c.Get(authRolesKey).(core.Roles)
		if !ok {
			c.Logger().Error("roles not extracted")
			return fmt.Errorf("roles not extracted")
		}
		if authUser, ok := c.Get(authUserKey).(models.User); ok && authUser.ID == user.ID {
			_ = v.addRoleByCode(c, roles, models.ObserveUserEmailRole)
			_ = v.addRoleByCode(c, roles, models.ObserveUserMiddleNameRole)
		}
		_ = v.addRoleByCode(c, roles, models.ObserveUserFirstNameRole)
		_ = v.addRoleByCode(c, roles, models.ObserveUserLastNameRole)
		return next(c)
	}
}

func (v *View) addRoleByCode(
	c echo.Context, roles core.Roles, code string,
) error {
	role, err := v.core.Roles.GetByCode(code)
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	roles[role.ID] = struct{}{}
	return nil
}

package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/labstack/echo"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

// fullUser represents user with additional information.
type fullUser struct {
	models.User
	// FirstName contains first name.
	FirstName string `json:",omitempty"`
	// LastName contains last name.
	LastName string `json:",omitempty"`
	// MiddleName contains middle name.
	MiddleName string `json:",omitempty"`
}

// registerUserHandlers registers handlers for user management.
func (v *View) registerUserHandlers(g *echo.Group) {
	g.GET(
		"/auth-status", v.authStatus,
		v.requireAuth(v.sessionAuth, v.guestAuth),
		v.requireRole(models.AuthStatusRole),
	)
	g.POST(
		"/login", v.loginUser,
		v.requireAuth(v.passwordAuth),
		v.requireRole(models.LoginRole),
	)
	g.POST(
		"/logout", v.logoutUser,
		v.requireAuth(v.sessionAuth),
		v.requireRole(models.LogoutRole),
	)
	g.POST(
		"/register", v.registerUser,
		v.requireAuth(v.guestAuth),
		v.requireRole(models.RegisterRole),
	)
}

// authStatus represents current authorization status.
type authStatus struct {
	User    *models.User    `json:",omitempty"`
	Session *models.Session `json:",omitempty"`
	Roles   []string        `json:""`
}

// authStatus returns current authorization status.
func (v *View) authStatus(c echo.Context) error {
	status := authStatus{}
	if session, ok := c.Get(authSessionKey).(models.Session); ok {
		status.Session = &session
		// Currently there can be no situation when user does not exist
		// with existing session, so panic should not happen.
		user := c.Get(authUserKey).(models.User)
		status.User = &user
	}
	for id := range c.Get(authRolesKey).(core.Roles) {
		if role, err := v.core.Roles.Get(id); err == nil {
			status.Roles = append(status.Roles, role.Code)
		}
	}
	return c.JSON(http.StatusOK, status)
}

// loginUser creates a new session for user.
func (v *View) loginUser(c echo.Context) error {
	user := c.Get(authUserKey).(models.User)
	expires := time.Now().Add(time.Hour * 24 * 90)
	session := models.Session{
		UserID:     user.ID,
		ExpireTime: expires.Unix(),
	}
	if err := session.GenerateSecret(); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := v.core.WithTx(func(tx *sql.Tx) error {
		var err error
		session, err = v.core.Sessions.CreateTx(tx, session)
		return err
	}); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	cookie := session.Cookie()
	cookie.Name = sessionCookie
	c.SetCookie(&cookie)
	return c.JSON(http.StatusCreated, session)
}

// logoutUser removes current session.
func (v *View) logoutUser(c echo.Context) error {
	session := c.Get(authSessionKey).(models.Session)
	if err := v.core.WithTx(func(tx *sql.Tx) error {
		return v.core.Sessions.DeleteTx(tx, session.ID)
	}); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusOK)
}

// registerUserForm represents form for register new user.
type registerUserForm struct {
	Login      string `json:""`
	Email      string `json:""`
	Password   string `json:""`
	FirstName  string `json:""`
	LastName   string `json:""`
	MiddleName string `json:""`
}

// registerUser registers user.
func (v *View) registerUser(c echo.Context) error {
	var form registerUserForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusBadRequest)
	}
	user := fullUser{
		User:       models.User{Login: form.Login},
		FirstName:  form.FirstName,
		LastName:   form.LastName,
		MiddleName: form.MiddleName,
	}
	if err := v.core.Users.SetPassword(
		&user.User, form.Password,
	); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := v.core.WithTx(func(tx *sql.Tx) error {
		var err error
		user.User, err = v.core.Users.CreateTx(tx, user.User)
		if err != nil {
			return err
		}
		return v.registerUserFields(tx, user.User, form)
	}); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, user)
}

// registerUserFields creates fields for registered user.
func (v *View) registerUserFields(
	tx *sql.Tx, user models.User, form registerUserForm,
) error {
	email := models.UserField{
		UserID: user.ID,
		Type:   models.EmailField,
		Data:   form.Email,
	}
	if _, err := v.core.UserFields.CreateTx(tx, email); err != nil {
		return err
	}
	if form.FirstName != "" {
		field := models.UserField{
			UserID: user.ID,
			Type:   models.FirstNameField,
			Data:   form.FirstName,
		}
		if _, err := v.core.UserFields.CreateTx(tx, field); err != nil {
			return err
		}
	}
	if form.LastName != "" {
		field := models.UserField{
			UserID: user.ID,
			Type:   models.LastNameField,
			Data:   form.LastName,
		}
		if _, err := v.core.UserFields.CreateTx(tx, field); err != nil {
			return err
		}
	}
	if form.MiddleName != "" {
		field := models.UserField{
			UserID: user.ID,
			Type:   models.MiddleNameField,
			Data:   form.MiddleName,
		}
		if _, err := v.core.UserFields.CreateTx(tx, field); err != nil {
			return err
		}
	}
	return nil
}

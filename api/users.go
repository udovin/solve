package api

import (
	"database/sql"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

// FullUser represents user with additional information.
type FullUser struct {
	models.User
	// FirstName contains first name.
	FirstName string `json:",omitempty"`
	// LastName contains last name.
	LastName string `json:",omitempty"`
	// MiddleName contains middle name.
	MiddleName string `json:",omitempty"`
}

// registerUserHandlers registers handlers for user management.
func (s *Server) registerUserHandlers(g *echo.Group) {
	g.POST(
		"/login", s.loginUser,
		s.requireAuth(s.passwordAuth),
		s.requireRole("login"),
	)
	g.POST(
		"/logout", s.logoutUser,
		s.requireAuth(s.sessionAuth),
		s.requireRole("logout"),
	)
	g.POST(
		"/register", s.registerUser,
		s.requireAuth(s.guestAuth),
		s.requireRole("register"),
	)
}

// loginUser creates a new session for user.
func (s *Server) loginUser(c echo.Context) error {
	user := c.Get(userKey).(models.User)
	expires := time.Now().Add(time.Hour * 24 * 90)
	session := models.Session{
		UserID:     user.ID,
		ExpireTime: expires.Unix(),
	}
	if err := session.GenerateSecret(); err != nil {
		c.Logger().Error(err)
		return err
	}
	if err := s.app.Sessions.Create(&session); err != nil {
		c.Logger().Error(err)
		return err
	}
	c.SetCookie(&http.Cookie{
		Name:    sessionKey,
		Value:   session.FormatCookie(),
		Expires: expires,
	})
	return c.JSON(http.StatusCreated, session)
}

// logoutUser removes current session.
func (s *Server) logoutUser(c echo.Context) error {
	session := c.Get(sessionKey).(models.Session)
	if err := s.app.Sessions.Delete(session.ID); err != nil {
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
func (s *Server) registerUser(c echo.Context) error {
	var form registerUserForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusBadRequest)
	}
	user := FullUser{
		User:       models.User{Login: form.Login},
		FirstName:  form.FirstName,
		LastName:   form.LastName,
		MiddleName: form.MiddleName,
	}
	if err := s.app.Users.SetPassword(
		&user.User, form.Password,
	); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := s.app.WithTx(func(tx *sql.Tx) error {
		var err error
		user.User, err = s.app.Users.CreateTx(tx, user.User)
		if err != nil {
			return err
		}
		return s.registerUserFields(tx, user.User, form)
	}); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, user)
}

// registerUserFields creates fields for registered user.
func (s *Server) registerUserFields(
	tx *sql.Tx, user models.User, form registerUserForm,
) error {
	email := models.UserField{
		UserID: user.ID,
		Type:   models.EmailField,
		Data:   form.Email,
	}
	if _, err := s.app.UserFields.CreateTx(tx, email); err != nil {
		return err
	}
	if form.FirstName != "" {
		field := models.UserField{
			UserID: user.ID,
			Type:   models.FirstNameField,
			Data:   form.FirstName,
		}
		if _, err := s.app.UserFields.CreateTx(tx, field); err != nil {
			return err
		}
	}
	if form.LastName != "" {
		field := models.UserField{
			UserID: user.ID,
			Type:   models.LastNameField,
			Data:   form.LastName,
		}
		if _, err := s.app.UserFields.CreateTx(tx, field); err != nil {
			return err
		}
	}
	if form.MiddleName != "" {
		field := models.UserField{
			UserID: user.ID,
			Type:   models.MiddleNameField,
			Data:   form.MiddleName,
		}
		if _, err := s.app.UserFields.CreateTx(tx, field); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) GetUser(c echo.Context) error {
	userID, err := strconv.ParseInt(c.Param("UserID"), 10, 64)
	var user models.User
	if err != nil {
		user, err = s.app.Users.GetByLogin(c.Param("UserID"))
	} else {
		user, err = s.app.Users.Get(userID)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	result := FullUser{User: user}
	fields, err := s.app.UserFields.GetByUser(user.ID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	for _, field := range fields {
		switch field.Type {
		case models.FirstNameField:
			result.FirstName = field.Data
		case models.LastNameField:
			result.LastName = field.Data
		case models.MiddleNameField:
			result.MiddleName = field.Data
		}
	}
	return c.JSON(http.StatusOK, result)
}

func (s *Server) GetUserSessions(c echo.Context) error {
	userID, err := strconv.ParseInt(c.Param("UserID"), 10, 64)
	var user models.User
	if err != nil {
		user, err = s.app.Users.GetByLogin(c.Param("UserID"))
	} else {
		user, err = s.app.Users.Get(userID)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	thisUser, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if user.ID != thisUser.ID && !thisUser.IsSuper {
		return c.NoContent(http.StatusForbidden)
	}
	sessions, err := s.app.Sessions.GetByUser(user.ID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	sort.Sort(sessionModelSorter(sessions))
	return c.JSON(http.StatusOK, sessions)
}

// func (s *Server) UpdateUser(c echo.Context) error {
// 	userID, err := strconv.ParseInt(c.Param("UserID"), 10, 64)
// 	var user models.User
// 	if err != nil {
// 		user, err = s.app.Users.GetByLogin(c.Param("UserID"))
// 	} else {
// 		user, err = s.app.Users.Get(userID)
// 	}
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return c.NoContent(http.StatusNotFound)
// 		}
// 		c.Logger().Error(err)
// 		return c.NoContent(http.StatusInternalServerError)
// 	}
// 	performer, ok := c.Get(userKey).(models.User)
// 	if !ok {
// 		return c.NoContent(http.StatusForbidden)
// 	}
// 	if user.ID != performer.ID && !performer.IsSuper {
// 		return c.NoContent(http.StatusForbidden)
// 	}
// 	var userData struct {
// 		Password *string `json:""`
// 	}
// 	if err := c.Bind(&userData); err != nil {
// 		c.Logger().Error(err)
// 		return c.NoContent(http.StatusBadRequest)
// 	}
// 	if userData.Password != nil {
// 		if err := s.app.Users.SetPassword(
// 			&user, *userData.Password,
// 		); err != nil {
// 			c.Logger().Error(err)
// 			return c.NoContent(http.StatusInternalServerError)
// 		}
// 	}
// 	if err := s.app.Users.Update(&user); err != nil {
// 		c.Logger().Error(err)
// 		return c.NoContent(http.StatusInternalServerError)
// 	}
// 	return c.JSON(http.StatusOK, user)
// }
//
// func (s *Server) DeleteUser(c echo.Context) error {
// 	return c.NoContent(http.StatusNotImplemented)
// }

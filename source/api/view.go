package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo"

	"../core"
	"../models"
)

type View struct {
	app    *core.App
	server *echo.Echo
}

func Register(app *core.App, server *echo.Echo) {
	v := View{app: app, server: server}
	server.GET("/api/v0/ping", v.Ping)
	server.POST("/api/v0/user", v.CreateUser)
	server.PATCH("/api/v0/user/:UserID", v.UpdateUser)
	server.POST("/api/v0/session", v.CreateSession)
	server.PATCH("/api/v0/session/:SessionID", v.UpdateSession)
	server.POST("/api/v0/problem", v.CreateProblem)
	server.PATCH("/api/v0/problem/:ProblemID", v.UpdateProblem)
}

func (v *View) Ping(c echo.Context) error {
	return c.JSON(http.StatusOK, "pong")
}

func (v *View) CreateUser(c echo.Context) error {
	var userData struct {
		Login    string `json:""`
		Email    string `json:""`
		Password string `json:""`
	}
	if err := c.Bind(&userData); err != nil {
		return err
	}
	user := models.User{
		Login: userData.Login,
	}
	if err := user.SetPassword(
		userData.Password, v.app.PasswordSalt,
	); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := v.app.UserStore.Create(&user); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, user)
}

func (v *View) UpdateUser(c echo.Context) error {
	userID, err := strconv.ParseInt(c.Param("UserID"), 10, 60)
	if err != nil {
		return err
	}
	user, ok := v.app.UserStore.Get(userID)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	c.Logger().Error(user)
	var userData struct {
		Password *string `json:""`
	}
	if err := c.Bind(&userData); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusBadRequest)
	}
	if userData.Password != nil {
		if err := user.SetPassword(
			*userData.Password, v.app.PasswordSalt,
		); err != nil {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}
	if err := v.app.UserStore.Update(&user); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, user)
}

func (v *View) DeleteUser(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) CreateSession(c echo.Context) error {
	var authData struct {
		Login    string `json:""`
		Password string `json:""`
	}
	if err := c.Bind(&authData); err != nil {
		return err
	}
	user, ok := v.app.UserStore.GetByLogin(authData.Login)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	if !user.CheckPassword(authData.Password, v.app.PasswordSalt) {
		return c.NoContent(http.StatusForbidden)
	}
	session := models.Session{
		UserID: user.ID,
	}
	if err := session.GenerateSecret(); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := v.app.SessionStore.Create(&session); err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, session)
}

func (v *View) UpdateSession(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) DeleteSession(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) CreateProblem(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) UpdateProblem(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) DeleteProblem(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

package api

import (
	"net/http"

	"github.com/labstack/echo"

	"../models"
)

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

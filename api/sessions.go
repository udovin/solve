package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

type Session struct {
	models.Session
	User models.User `json:""`
}

func (v *View) GetSessions(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) CreateSession(c echo.Context) error {
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	expires := time.Now().Add(time.Hour * 24 * 90)
	session := models.Session{
		UserID:     user.ID,
		ExpireTime: expires.Unix(),
	}
	if err := session.GenerateSecret(); err != nil {
		c.Logger().Error(err)
		return err
	}
	if err := v.app.Sessions.Create(&session); err != nil {
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

func (v *View) UpdateSession(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (v *View) DeleteSession(c echo.Context) error {
	sessionID, err := strconv.ParseInt(c.Param("SessionID"), 10, 64)
	if err != nil {
		return err
	}
	user, ok := c.Get(userKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	session, ok := v.app.Sessions.Get(sessionID)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	if session.UserID != user.ID {
		return c.NoContent(http.StatusForbidden)
	}
	if err := v.app.Sessions.Delete(session.ID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusOK)
}

func (v *View) GetCurrentSession(c echo.Context) error {
	session, ok := c.Get(sessionKey).(models.Session)
	if !ok {
		return c.NoContent(http.StatusInternalServerError)
	}
	user, ok := v.app.Users.Get(session.UserID)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	return c.JSON(http.StatusOK, Session{
		Session: session,
		User:    user,
	})
}

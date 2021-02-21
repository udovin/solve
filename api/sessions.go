package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

// registerSessionHandlers registers handlers for session management.
func (v *View) registerSessionHandlers(g *echo.Group) {
	g.GET(
		"/sessions/:session", v.observeSession,
		v.sessionAuth, v.requireAuth, v.extractSession, v.extractSessionRoles,
		v.requireAuthRole(models.ObserveSessionRole),
	)
	g.DELETE(
		"/sessions/:session", v.deleteSession,
		v.sessionAuth, v.requireAuth, v.extractSession, v.extractSessionRoles,
		v.requireAuthRole(models.DeleteSessionRole),
	)
}

func (v *View) observeSession(c echo.Context) error {
	session, ok := c.Get(sessionKey).(models.Session)
	if !ok {
		c.Logger().Error("session not extracted")
		return fmt.Errorf("session not extracted")
	}
	resp := Session{
		ID:         session.ID,
		CreateTime: session.CreateTime,
		ExpireTime: session.ExpireTime,
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) deleteSession(c echo.Context) error {
	session, ok := c.Get(sessionKey).(models.Session)
	if !ok {
		c.Logger().Error("session not extracted")
		return fmt.Errorf("session not extracted")
	}
	if err := v.core.WithTx(c.Request().Context(), func(tx *sql.Tx) error {
		return v.core.Sessions.DeleteTx(tx, session.ID)
	}); err != nil {
		c.Logger().Error(err)
		return err
	}
	resp := Session{
		ID:         session.ID,
		CreateTime: session.CreateTime,
		ExpireTime: session.ExpireTime,
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) extractSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("session"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return err
		}
		session, err := v.core.Sessions.Get(id)
		if err != nil {
			if err == sql.ErrNoRows {
				resp := errorResp{Message: "session not found"}
				return c.JSON(http.StatusNotFound, resp)
			}
			c.Logger().Error(err)
			return err
		}
		c.Set(sessionKey, session)
		return next(c)
	}
}

func (v *View) extractSessionRoles(next echo.HandlerFunc) echo.HandlerFunc {
	nextWrap := func(c echo.Context) error {
		session, ok := c.Get(sessionKey).(models.Session)
		if !ok {
			c.Logger().Error("session not extracted")
			return fmt.Errorf("session not extracted")
		}
		roles, ok := c.Get(authRolesKey).(core.RoleSet)
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
		if ok && authUser.AccountID == session.AccountID {
			addRole(roles, models.ObserveSessionRole)
			addRole(roles, models.DeleteSessionRole)
		}
		return next(c)
	}
	return v.extractAuthRoles(nextWrap)
}

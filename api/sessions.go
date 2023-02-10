package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

// Session represents session.
type Session struct {
	// ID contains session ID.
	ID int64 `json:"id"`
	// CreateTime contains session create time.
	CreateTime int64 `json:"create_time,omitempty"`
	// ExpireTime contains session expire time.
	ExpireTime int64 `json:"expire_time,omitempty"`
}

// Sessions represents sessions response.
type Sessions struct {
	Sessions []Session `json:"sessions"`
}

// registerSessionHandlers registers handlers for session management.
func (v *View) registerSessionHandlers(g *echo.Group) {
	g.GET(
		"/v0/sessions/:session", v.observeSession,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractSession,
		v.requirePermission(models.ObserveSessionRole),
	)
	g.DELETE(
		"/v0/sessions/:session", v.deleteSession,
		v.extractAuth(v.sessionAuth), v.extractSession,
		v.requirePermission(models.DeleteSessionRole),
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
	if err := v.core.Sessions.Delete(getContext(c), session.ID); err != nil {
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
		if err := syncStore(c, v.core.Sessions); err != nil {
			return err
		}
		session, err := v.core.Sessions.Get(getContext(c), id)
		if err != nil {
			if err == sql.ErrNoRows {
				resp := errorResponse{
					Message: localize(c, "Session not found."),
				}
				return c.JSON(http.StatusNotFound, resp)
			}
			c.Logger().Error(err)
			return err
		}
		accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
		if !ok {
			c.Logger().Error("auth not extracted")
			return fmt.Errorf("auth not extracted")
		}
		c.Set(sessionKey, session)
		c.Set(permissionCtxKey, v.getSessionPermissions(accountCtx, session))
		return next(c)
	}
}

func (v *View) getSessionPermissions(
	ctx *managers.AccountContext, session models.Session,
) managers.PermissionSet {
	permissions := ctx.Permissions.Clone()
	if account := ctx.Account; account != nil && account.ID == session.AccountID {
		permissions[models.ObserveSessionRole] = struct{}{}
		permissions[models.DeleteSessionRole] = struct{}{}
	}
	return permissions
}

package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/labstack/echo"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

// View represents API view.
type View struct {
	core *core.Core
}

// Register registers handlers in specified group.
func (v *View) Register(g *echo.Group) {
	g.Use(v.logVisit)
	g.GET("/ping", v.ping)
	v.registerUserHandlers(g)
}

// ping returns pong.
func (v *View) ping(c echo.Context) error {
	return c.JSON(http.StatusOK, "pong")
}

// NewView returns a new instance of view.
func NewView(core *core.Core) *View {
	return &View{core: core}
}

const (
	authUserKey    = "AuthUser"
	authSessionKey = "AuthSession"
	authVisitKey   = "AuthVisit"
	authRolesKey   = "AuthRoles"
	sessionCookie  = "session"
)

// logVisit saves visit to visit store.
func (v *View) logVisit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set(authVisitKey, v.core.Visits.MakeFromContext(c))
		defer func() {
			visit := c.Get(authVisitKey).(models.Visit)
			if user, ok := c.Get(authUserKey).(models.User); ok {
				visit.UserID = models.NInt64(user.ID)
			}
			if session, ok := c.Get(authSessionKey).(models.Session); ok {
				visit.SessionID = models.NInt64(session.ID)
			}
			visit.Status = c.Response().Status
			if err := v.core.WithTx(func(tx *sql.Tx) error {
				_, err := v.core.Visits.CreateTx(tx, visit)
				return err
			}); err != nil {
				c.Logger().Error(err)
			}
		}()
		return next(c)
	}
}

type authMethod func(echo.Context) error

var errNoAuth = errors.New("bad auth")

func (v *View) requireAuth(methods ...authMethod) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			for _, method := range methods {
				if err := method(c); err != nil {
					if err == errNoAuth {
						continue
					}
					c.Logger().Error(err)
					return err
				}
				if err := v.extractRoles(c); err != nil {
					c.Logger().Error(err)
					return err
				}
				return next(c)
			}
			return c.NoContent(http.StatusForbidden)
		}
	}
}

// extractRoles extract roles for user.
func (v *View) extractRoles(c echo.Context) error {
	if user, ok := c.Get(authUserKey).(models.User); ok {
		roles, err := v.core.GetUserRoles(user.ID)
		if err != nil {
			return err
		}
		c.Set(authRolesKey, roles)
	} else {
		roles, err := v.core.GetGuestRoles()
		if err != nil {
			return err
		}
		c.Set(authRolesKey, roles)
	}
	return nil
}

// guestAuth authorizes guest.
func (v *View) guestAuth(c echo.Context) error {
	return nil
}

// getSessionByCookie returns session.
func (v *View) getSessionByCookie(value string) (models.Session, error) {
	session, err := v.core.Sessions.GetByCookie(value)
	if err == sql.ErrNoRows {
		if err := v.core.WithTx(v.core.Sessions.SyncTx); err != nil {
			return models.Session{}, err
		}
		session, err = v.core.Sessions.GetByCookie(value)
	}
	if err != nil {
		return models.Session{}, err
	}
	return session, nil
}

// sessionAuth tries to auth using session cookie.
func (v *View) sessionAuth(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil {
		return errNoAuth
	}
	session, err := v.getSessionByCookie(cookie.Value)
	if err != nil {
		return errNoAuth
	}
	user, err := v.core.Users.Get(session.UserID)
	if err != nil {
		return errNoAuth
	}
	c.Set(authUserKey, user)
	c.Set(authSessionKey, session)
	return nil
}

// passwordAuth tries to auth using login and password.
func (v *View) passwordAuth(c echo.Context) error {
	var authData struct {
		Login    string `json:""`
		Password string `json:""`
	}
	if err := c.Bind(&authData); err != nil {
		return errNoAuth
	}
	user, err := v.core.Users.GetByLogin(authData.Login)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNoAuth
		}
		return err
	}
	if !v.core.Users.CheckPassword(user, authData.Password) {
		return errNoAuth
	}
	c.Set(authUserKey, user)
	return nil
}

// requireRole check that user has required roles.
func (v *View) requireRole(code string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			roles, ok := c.Get(authRolesKey).(core.Roles)
			if !ok {
				return c.NoContent(http.StatusForbidden)
			}
			ok, err := v.core.HasRole(roles, code)
			if err != nil {
				return c.NoContent(http.StatusInternalServerError)
			}
			if !ok {
				return c.NoContent(http.StatusForbidden)
			}
			return next(c)
		}
	}
}

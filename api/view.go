package api

import (
	"context"
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
	g.GET("/health", v.health)
	v.registerUserHandlers(g)
	v.registerRoleHandlers(g)
}

// ping returns pong.
func (v *View) ping(c echo.Context) error {
	return c.String(http.StatusOK, "pong")
}

// health returns current healthiness status.
func (v *View) health(c echo.Context) error {
	return c.JSON(http.StatusOK, nil)
}

// NewView returns a new instance of view.
func NewView(core *core.Core) *View {
	return &View{core: core}
}

const (
	authAccountKey = "auth_account"
	authSessionKey = "auth_session"
	authVisitKey   = "auth_visit"
	authRolesKey   = "auth_roles"
	authUserKey    = "auth_user"
	userKey        = "user"
	sessionCookie  = "session"
)

// logVisit saves visit to visit store.
func (v *View) logVisit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set(authVisitKey, v.core.Visits.MakeFromContext(c))
		defer func() {
			visit := c.Get(authVisitKey).(models.Visit)
			if account, ok := c.Get(authAccountKey).(models.Account); ok {
				visit.AccountID = models.NInt64(account.ID)
			}
			if session, ok := c.Get(authSessionKey).(models.Session); ok {
				visit.SessionID = models.NInt64(session.ID)
			}
			visit.Status = c.Response().Status
			if err := v.core.WithTx(
				c.Request().Context(),
				func(tx *sql.Tx) error {
					_, err := v.core.Visits.CreateTx(tx, visit)
					return err
				},
			); err != nil {
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
	if account, ok := c.Get(authAccountKey).(models.Account); ok {
		roles, err := v.core.GetAccountRoles(account.ID)
		if err != nil {
			return err
		}
		if len(roles) == 0 && account.Kind == models.UserAccount {
			roles, err = v.core.GetUserRoles()
			if err != nil {
				return err
			}
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
func (v *View) getSessionByCookie(
	ctx context.Context, value string,
) (models.Session, error) {
	session, err := v.core.Sessions.GetByCookie(value)
	if err == sql.ErrNoRows {
		if err := v.core.WithTx(ctx, v.core.Sessions.SyncTx); err != nil {
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
	session, err := v.getSessionByCookie(c.Request().Context(), cookie.Value)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNoAuth
		}
		return err
	}
	account, err := v.core.Accounts.Get(session.AccountID)
	if err != nil {
		return err
	}
	if account.Kind != models.UserAccount {
		c.Logger().Errorf(
			"Account %v should have %v kind, but has %v",
			account.ID, models.UserAccount, account.Kind,
		)
		return errNoAuth
	}
	user, err := v.core.Users.GetByAccount(session.AccountID)
	if err != nil {
		return err
	}
	c.Set(authAccountKey, account)
	c.Set(authUserKey, user)
	c.Set(authSessionKey, session)
	return nil
}

type userAuthForm struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// userAuth tries to auth using user login and password.
func (v *View) userAuth(c echo.Context) error {
	var form userAuthForm
	if err := c.Bind(&form); err != nil {
		return errNoAuth
	}
	user, err := v.core.Users.GetByLogin(form.Login)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNoAuth
		}
		return err
	}
	if !v.core.Users.CheckPassword(user, form.Password) {
		return errNoAuth
	}
	account, err := v.core.Accounts.Get(user.AccountID)
	if err != nil {
		return err
	}
	if account.Kind != models.UserAccount {
		c.Logger().Errorf(
			"Account %v should have %v kind, but has %v",
			account.ID, models.UserAccount, account.Kind,
		)
		return errNoAuth
	}
	c.Set(authAccountKey, account)
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
				c.Logger().Error(err)
				return c.NoContent(http.StatusInternalServerError)
			}
			if !ok {
				return c.NoContent(http.StatusForbidden)
			}
			return next(c)
		}
	}
}

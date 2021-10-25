package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

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
	v.registerSessionHandlers(g)
	v.registerContestHandlers(g)
	v.registerProblemHandlers(g)
}

func (v *View) RegisterSocket(g *echo.Group) {
	g.GET("/ping", v.ping)
	g.GET("/health", v.health)
	v.registerSocketUserHandlers(g)
	v.registerSocketRoleHandlers(g)
}

// ping returns pong.
func (v *View) ping(c echo.Context) error {
	return c.String(http.StatusOK, "pong")
}

// health returns current healthiness status.
func (v *View) health(c echo.Context) error {
	if err := v.core.DB.Ping(); err != nil {
		c.Logger().Error(err)
		return c.String(http.StatusInternalServerError, "unhealthy")
	}
	return c.String(http.StatusOK, "healthy")
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
	roleKey        = "role"
	userKey        = "user"
	sessionKey     = "session"
	sessionCookie  = "session"
	contestKey     = "contest"
	problemKey     = "problem"
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

type errorField struct {
	Message string `json:"message"`
}

type errorFields map[string]errorField

type errorResp struct {
	// Message.
	Message string `json:"message"`
	// MissingRoles.
	MissingRoles []string `json:"missing_roles,omitempty"`
	// InvalidFields.
	InvalidFields errorFields `json:"invalid_fields"`
}

// Error returns response error message.
func (r errorResp) Error() string {
	return r.Message
}

// sessionAuth tries to authorize account by session.
func (v *View) sessionAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if _, ok := c.Get(authAccountKey).(models.Account); ok {
			return next(c)
		}
		cookie, err := c.Cookie(sessionCookie)
		if err != nil {
			if err == http.ErrNoCookie {
				return next(c)
			}
			c.Logger().Warn(err)
			return err
		}
		session, err := v.getSessionByCookie(
			c.Request().Context(), cookie.Value,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				return next(c)
			}
			c.Logger().Error(err)
			return err
		}
		account, err := v.core.Accounts.Get(session.AccountID)
		if err != nil {
			c.Logger().Error(err)
			return err
		}
		if account.Kind != models.UserAccount {
			resp := errorResp{Message: "only user account supported"}
			return c.JSON(http.StatusNotImplemented, resp)
		}
		user, err := v.core.Users.GetByAccount(session.AccountID)
		if err != nil {
			c.Logger().Error(err)
			return err
		}
		c.Set(authAccountKey, account)
		c.Set(authUserKey, user)
		c.Set(authSessionKey, session)
		return next(c)
	}
}

type userAuthForm struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// userAuth tries to authorize user by login and password.
func (v *View) userAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if _, ok := c.Get(authAccountKey).(models.Account); ok {
			return next(c)
		}
		var form userAuthForm
		if err := c.Bind(&form); err != nil {
			return err
		}
		if form.Login == "" || form.Password == "" {
			return next(c)
		}
		user, err := v.core.Users.GetByLogin(form.Login)
		if err != nil {
			if err == sql.ErrNoRows {
				resp := errorResp{Message: "user not found"}
				return c.JSON(http.StatusForbidden, resp)
			}
			c.Logger().Error(err)
			return err
		}
		if !v.core.Users.CheckPassword(user, form.Password) {
			resp := errorResp{Message: "user not found"}
			return c.JSON(http.StatusForbidden, resp)
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
			return fmt.Errorf("invalid account kind %q", account.Kind)
		}
		c.Set(authAccountKey, account)
		c.Set(authUserKey, user)
		return next(c)
	}
}

// requireAuth checks account authorization.
func (v *View) requireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if _, ok := c.Get(authAccountKey).(models.Account); !ok {
			resp := errorResp{Message: "auth required"}
			return c.JSON(http.StatusForbidden, resp)
		}
		return next(c)
	}
}

// extractAuthRoles extract roles for user.
func (v *View) extractAuthRoles(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if _, ok := c.Get(authRolesKey).(core.RoleSet); ok {
			return next(c)
		}
		if account, ok := c.Get(authAccountKey).(models.Account); ok {
			roles, err := v.core.GetAccountRoles(account.ID)
			if err != nil {
				c.Logger().Error(err)
				return err
			}
			c.Set(authRolesKey, roles)
		} else {
			roles, err := v.core.GetGuestRoles()
			if err != nil {
				c.Logger().Error(err)
				return err
			}
			c.Set(authRolesKey, roles)
		}
		return next(c)
	}
}

// requireRole check that user has required roles.
func (v *View) requireAuthRole(codes ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		nextWrap := func(c echo.Context) error {
			resp := errorResp{Message: "account missing roles"}
			roles, ok := c.Get(authRolesKey).(core.RoleSet)
			if !ok {
				resp.MissingRoles = codes
				return c.JSON(http.StatusForbidden, resp)
			}
			for _, code := range codes {
				ok, err := v.core.HasRole(roles, code)
				if err != nil {
					c.Logger().Error(err)
					return err
				}
				if !ok {
					resp.MissingRoles = append(resp.MissingRoles, code)
				}
			}
			if len(resp.MissingRoles) > 0 {
				return c.JSON(http.StatusForbidden, resp)
			}
			return next(c)
		}
		return v.extractAuthRoles(nextWrap)
	}
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

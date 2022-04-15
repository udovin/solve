package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/udovin/gosql"
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
	v.registerSolutionHandlers(g)
	v.registerCompilerHandlers(g)
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
	authAccountKey        = "auth_account"
	authSessionKey        = "auth_session"
	authVisitKey          = "auth_visit"
	authRolesKey          = "auth_roles"
	authUserKey           = "auth_user"
	roleKey               = "role"
	childRoleKey          = "child_role"
	userKey               = "user"
	sessionKey            = "session"
	sessionCookie         = "session"
	contestKey            = "contest"
	contestProblemKey     = "contest_problem"
	contestParticipantKey = "contest_participant"
	contestSolutionKey    = "contest_solution"
	problemKey            = "problem"
	solutionKey           = "solution"
	compilerKey           = "compiler"
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
			if s := v.getBoolSetting(c, "log_visit."+c.Path()); s == nil || *s {
				if err := v.core.Visits.Create(c.Request().Context(), &visit); err != nil {
					c.Logger().Error(err)
				}
			}
		}()
		return next(c)
	}
}

type errorField struct {
	Message string `json:"message"`
}

type errorFields map[string]errorField

type errorResponse struct {
	// Message.
	Message string `json:"message"`
	// MissingPermissions.
	MissingPermissions []string `json:"missing_permissions,omitempty"`
	// InvalidFields.
	InvalidFields errorFields `json:"invalid_fields"`
}

// Error returns response error message.
func (r errorResponse) Error() string {
	var result strings.Builder
	result.WriteString(r.Message)
	if len(r.MissingPermissions) > 0 {
		result.WriteString(" (missing permissions: ")
		for i, role := range r.MissingPermissions {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(role)
		}
		result.WriteRune(')')
	}
	return result.String()
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
			resp := errorResponse{
				Message: "only user account supported",
			}
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
				resp := errorResponse{
					Message: "user not found",
				}
				return c.JSON(http.StatusForbidden, resp)
			}
			c.Logger().Error(err)
			return err
		}
		if !v.core.Users.CheckPassword(user, form.Password) {
			resp := errorResponse{
				Message: "user not found",
			}
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
			resp := errorResponse{
				Message: "auth required",
			}
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
func (v *View) requireAuthRole(names ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		nextWrap := func(c echo.Context) error {
			resp := errorResponse{
				Message: "account missing permissions",
			}
			roles, ok := c.Get(authRolesKey).(core.RoleSet)
			if !ok {
				resp.MissingPermissions = names
				return c.JSON(http.StatusForbidden, resp)
			}
			for _, name := range names {
				ok, err := v.core.HasRole(roles, name)
				if err != nil {
					c.Logger().Error(err)
					return err
				}
				if !ok {
					resp.MissingPermissions = append(resp.MissingPermissions, name)
				}
			}
			if len(resp.MissingPermissions) > 0 {
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
		if err := v.core.Sessions.Sync(ctx); err != nil {
			return models.Session{}, err
		}
		session, err = v.core.Sessions.GetByCookie(value)
	}
	if err != nil {
		return models.Session{}, err
	}
	return session, nil
}

var (
	truePtr  = getPtr(true)
	falsePtr = getPtr(false)
)

func (v *View) getBoolSetting(ctx echo.Context, key string) *bool {
	setting, err := v.core.Settings.GetByKey(key)
	if err != nil {
		if err != sql.ErrNoRows {
			ctx.Logger().Error("Error:", err)
		}
		return nil
	}
	switch strings.ToLower(setting.Value) {
	case "1", "t", "true":
		return truePtr
	case "0", "f", "false":
		return falsePtr
	default:
		ctx.Logger().Warnf(
			"Setting %q has invalid value %q",
			key, setting.Value,
		)
		return nil
	}
}

func getPtr[T any](object T) *T {
	return &object
}

var (
	sqlRepeatableRead = gosql.WithIsolation(sql.LevelRepeatableRead)
	sqlReadOnly       = gosql.WithReadOnly(true)
)

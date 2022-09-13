package api

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

// View represents API view.
type View struct {
	core     *core.Core
	accounts *managers.AccountManager
	contests *managers.ContestManager
	files    *managers.FileManager
}

// Register registers handlers in specified group.
func (v *View) Register(g *echo.Group) {
	g.Use(wrapResponse, v.logVisit)
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
	g.Use(wrapResponse, v.extractAuth(v.guestAuth))
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
	v := View{
		core:     core,
		accounts: managers.NewAccountManager(core),
		contests: managers.NewContestManager(core),
	}
	if core.Config.Storage != nil {
		v.files = managers.NewFileManager(core)
	}
	return &v
}

const (
	nowKey                = "now"
	authVisitKey          = "auth_visit"
	authSessionKey        = "auth_session"
	accountCtxKey         = "account_ctx"
	permissionCtxKey      = "permission_ctx"
	roleKey               = "role"
	childRoleKey          = "child_role"
	userKey               = "user"
	sessionKey            = "session"
	sessionCookie         = "session"
	contestCtxKey         = "contest_ctx"
	contestProblemKey     = "contest_problem"
	contestParticipantKey = "contest_participant"
	contestSolutionKey    = "contest_solution"
	problemKey            = "problem"
	solutionKey           = "solution"
	compilerKey           = "compiler"
)

type (
	NInt64     = models.NInt64
	NString    = models.NString
	FileReader = managers.FileReader
)

type JSON struct {
	models.JSON
}

func (v *JSON) UnmarshalParam(data string) error {
	return v.JSON.UnmarshalJSON([]byte(data))
}

// logVisit saves visit to visit store.
func (v *View) logVisit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set(authVisitKey, v.core.Visits.MakeFromContext(c))
		defer func() {
			visit := c.Get(authVisitKey).(models.Visit)
			if ctx, ok := c.Get(accountCtxKey).(*managers.AccountContext); ok {
				if ctx.Account != nil {
					visit.AccountID = NInt64(ctx.Account.ID)
				}
			}
			if session, ok := c.Get(authSessionKey).(models.Session); ok {
				visit.SessionID = NInt64(session.ID)
			}
			visit.Status = c.Response().Status
			if s := v.getBoolSetting(c, "log_visit."+c.Path()); s == nil || *s {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				if err := v.core.Visits.Create(ctx, &visit); err != nil {
					c.Logger().Error("Unable to create visit", err)
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
	// Code.
	Code int `json:"-"`
	// Message.
	Message string `json:"message"`
	// MissingPermissions.
	MissingPermissions []string `json:"missing_permissions,omitempty"`
	// InvalidFields.
	InvalidFields errorFields `json:"invalid_fields,omitempty"`
}

// StatusCode returns response status code.
func (r errorResponse) StatusCode() int {
	return r.Code
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
	if len(r.InvalidFields) > 0 {
		result.WriteString(" (invalid fields: ")
		i := 0
		for field := range r.InvalidFields {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(field)
			i++
		}
		result.WriteRune(')')
	}
	return result.String()
}

type statusCodeResponse interface {
	StatusCode() int
}

var rnd = rand.NewSource(time.Now().UnixNano())

func wrapResponse(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		reqID := c.Request().Header.Get(echo.HeaderXRequestID)
		if reqID == "" {
			reqID = fmt.Sprintf("%d-%d", rnd.Int63(), time.Now().UnixMilli())
		}
		logger := c.Logger().(*core.Logger).With(core.Any("req_id", reqID))
		c.SetLogger(logger)
		c.Response().Header().Add(echo.HeaderXRequestID, reqID)
		c.Response().Header().Add("X-Solve-Version", config.Version)
		start := time.Now()
		err := next(c)
		defer func() {
			finish := time.Now()
			status := c.Response().Status
			message := fmt.Sprintf("%s %s", c.Request().Method, c.Request().RequestURI)
			params := map[string]string{}
			for _, name := range c.ParamNames() {
				params[name] = c.Param(name)
			}
			args := []any{
				message,
				core.Any("status", status),
				core.Any("method", c.Request().Method),
				core.Any("path", c.Path()),
				core.Any("params", params),
				core.Any("remote_ip", c.RealIP()),
				core.Any("latency", finish.Sub(start)),
				err,
			}
			switch {
			case status >= 500:
				logger.Error(args...)
			case status >= 400:
				logger.Warn(args...)
			default:
				logger.Info(args...)
			}
		}()
		if resp, ok := err.(statusCodeResponse); ok {
			code := resp.StatusCode()
			if code == 0 {
				code = http.StatusInternalServerError
			}
			return c.JSON(code, resp)
		}
		return err
	}
}

type authMethod func(c echo.Context) (bool, error)

func (v *View) extractAuth(authMethods ...authMethod) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			for _, method := range authMethods {
				ok, err := method(c)
				if err != nil {
					c.Logger().Error(err)
					return err
				}
				if ok {
					return next(c)
				}
			}
			resp := errorResponse{
				Code:    http.StatusForbidden,
				Message: "unable to authorize",
			}
			return c.JSON(http.StatusForbidden, resp)
		}
	}
}

func (v *View) sessionAuth(c echo.Context) (bool, error) {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil {
		if err == http.ErrNoCookie {
			return false, nil
		}
		return false, err
	}
	if len(cookie.Value) == 0 {
		return false, nil
	}
	session, err := v.getSessionByCookie(getContext(c), cookie.Value)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	account, err := v.core.Accounts.Get(session.AccountID)
	if err != nil {
		return false, err
	}
	accountCtx, err := v.accounts.MakeContext(getContext(c), &account)
	if err != nil {
		return false, err
	}
	c.Set(authSessionKey, session)
	c.Set(accountCtxKey, accountCtx)
	c.Set(permissionCtxKey, accountCtx)
	return true, nil
}

type userAuthForm struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (v *View) userAuth(c echo.Context) (bool, error) {
	var form userAuthForm
	if err := c.Bind(&form); err != nil {
		return false, err
	}
	if form.Login == "" || form.Password == "" {
		return false, nil
	}
	user, err := v.core.Users.GetByLogin(form.Login)
	if err != nil {
		if err == sql.ErrNoRows {
			resp := errorResponse{
				Code:    http.StatusForbidden,
				Message: "user not found",
			}
			return false, resp
		}
		return false, err
	}
	if !v.core.Users.CheckPassword(user, form.Password) {
		resp := errorResponse{
			Code:    http.StatusForbidden,
			Message: "invalid password",
		}
		return false, resp
	}
	account, err := v.core.Accounts.Get(user.AccountID)
	if err != nil {
		return false, err
	}
	if account.Kind != models.UserAccount {
		c.Logger().Errorf(
			"Account %v should have %v kind, but has %v",
			account.ID, models.UserAccount, account.Kind,
		)
		return false, fmt.Errorf("invalid account kind %q", account.Kind)
	}
	accountCtx, err := v.accounts.MakeContext(getContext(c), &account)
	if err != nil {
		return false, err
	}
	c.Set(accountCtxKey, accountCtx)
	c.Set(permissionCtxKey, accountCtx)
	return true, nil
}

func (v *View) guestAuth(c echo.Context) (bool, error) {
	ctx, err := v.accounts.MakeContext(getContext(c), nil)
	if err != nil {
		return false, err
	}
	c.Set(accountCtxKey, ctx)
	c.Set(permissionCtxKey, ctx)
	return true, nil
}

// requireRole check that user has required roles.
func (v *View) requirePermission(names ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			resp := errorResponse{
				Message: "account missing permissions",
			}
			ctx, ok := c.Get(permissionCtxKey).(managers.Permissions)
			if !ok {
				resp.MissingPermissions = names
				return c.JSON(http.StatusForbidden, resp)
			}
			for _, name := range names {
				if !ctx.HasPermission(name) {
					resp.MissingPermissions = append(resp.MissingPermissions, name)
				}
			}
			if len(resp.MissingPermissions) > 0 {
				return c.JSON(http.StatusForbidden, resp)
			}
			return next(c)
		}
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
			ctx.Logger().Error(
				"Unable to get setting",
				core.Any("key", key), err,
			)
		}
		return nil
	}
	switch strings.ToLower(setting.Value) {
	case "1", "t", "true":
		return truePtr
	case "0", "f", "false":
		return falsePtr
	default:
		ctx.Logger().Warn(
			"Setting has invalid value",
			core.Any("key", key),
			core.Any("value", setting.Value),
		)
		return nil
	}
}

func getContext(c echo.Context) context.Context {
	ctx := c.Request().Context()
	if t, ok := c.Get(nowKey).(time.Time); ok {
		ctx = models.WithNow(ctx, t)
	}
	accountCtx, ok := c.Get(accountCtxKey).(*managers.AccountContext)
	if ok && accountCtx.Account != nil {
		ctx = models.WithAccountID(ctx, accountCtx.Account.ID)
	}
	return ctx
}

func getNow(c echo.Context) time.Time {
	t, ok := c.Get(nowKey).(time.Time)
	if !ok {
		return time.Now()
	}
	return t
}

func getPtr[T any](object T) *T {
	return &object
}

var (
	sqlRepeatableRead = gosql.WithIsolation(sql.LevelRepeatableRead)
	sqlReadOnly       = gosql.WithReadOnly(true)
)

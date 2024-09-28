package api

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/labstack/echo/v4"
	"golang.org/x/text/language"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/config"
	"github.com/udovin/solve/internal/core"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
	"github.com/udovin/solve/internal/pkg/logs"
)

// View represents API view.
type View struct {
	core      *core.Core
	accounts  *managers.AccountManager
	contests  *managers.ContestManager
	files     *managers.FileManager
	solutions *managers.SolutionManager
	standings *managers.ContestStandingsManager
	visits    chan visitContext
}

// Register registers handlers in specified group.
func (v *View) Register(g *echo.Group) {
	g.Use(wrapResponse, v.wrapSyncStores, v.logVisit, v.extractLocale)
	g.GET("/ping", v.ping)
	g.GET("/health", v.health)
	v.registerAccountHandlers(g)
	v.registerUserHandlers(g)
	v.registerScopeHandlers(g)
	v.registerGroupHandlers(g)
	v.registerRoleHandlers(g)
	v.registerSessionHandlers(g)
	v.registerTokenHandlers(g)
	v.registerContestHandlers(g)
	v.registerContestStandingsHandlers(g)
	v.registerContestMessageHandlers(g)
	v.registerProblemHandlers(g)
	v.registerSolutionHandlers(g)
	v.registerCompilerHandlers(g)
	v.registerSettingHandlers(g)
	v.registerLocaleHandlers(g)
	v.registerFileHandlers(g)
	v.registerTokenHandlers(g)
}

func (v *View) RegisterSocket(g *echo.Group) {
	g.Use(wrapResponse, v.wrapSyncStores, v.extractAuth(v.guestAuth))
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
		core:      core,
		accounts:  managers.NewAccountManager(core),
		contests:  managers.NewContestManager(core),
		standings: managers.NewContestStandingsManager(core),
	}
	if core.Config.Storage != nil {
		v.files = managers.NewFileManager(core)
	}
	if v.files != nil {
		v.solutions = managers.NewSolutionManager(core, v.files)
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
	fileKey               = "file"
	settingKey            = "setting"
	scopeKey              = "scope"
	scopeUserKey          = "scope_user"
	groupKey              = "group"
	groupMemberKey        = "group_member"
	tokenKey              = "token"
	localeKey             = "locale"
	syncKey               = "sync"
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
			select {
			case v.visits <- visitContext{
				Path:   c.Path(),
				Visit:  visit,
				Logger: c.Logger(),
			}:
			default:
				c.Logger().Error("Visits queue overflow")
			}
		}()
		return next(c)
	}
}

func isValidLocaleName(name string) bool {
	return name == "en" || name == "ru"
}

func (v *View) extractLocale(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if cookie, err := c.Cookie("locale"); err == nil {
			if name := cookie.Value; isValidLocaleName(name) {
				locale := settingLocale{
					name:     name,
					settings: v.core.Settings,
				}
				c.Set(localeKey, &locale)
				return next(c)
			}
		}
		acceptLanguage := c.Request().Header.Get("Accept-Language")
		tags, _, err := language.ParseAcceptLanguage(acceptLanguage)
		if err != nil {
			return next(c)
		}
		for _, tag := range tags {
			if name := tag.String(); isValidLocaleName(name) {
				locale := settingLocale{
					name:     name,
					settings: v.core.Settings,
				}
				c.Set(localeKey, &locale)
				return next(c)
			}
		}
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

var (
	rnd      = rand.NewSource(time.Now().UnixNano())
	rndMutex = sync.Mutex{}
)

func randUint32() uint32 {
	rndMutex.Lock()
	defer rndMutex.Unlock()
	return uint32(rnd.Int63() >> 32)
}

func wrapResponse(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		reqID := c.Request().Header.Get(echo.HeaderXRequestID)
		if reqID == "" {
			reqID = fmt.Sprintf("%d-%d", time.Now().UnixMilli(), randUint32())
		}
		logger := c.Logger().(*logs.Logger).With(logs.Any("req_id", reqID))
		c.SetLogger(logger)
		c.Response().Header().Add(echo.HeaderXRequestID, reqID)
		c.Response().Header().Add("X-Solve-Version", config.Version)
		start := time.Now()
		err := next(c)
		status := c.Response().Status
		if err != nil {
			status = 500
		}
		defer func() {
			finish := time.Now()
			message := fmt.Sprintf("%s %s", c.Request().Method, c.Request().RequestURI)
			params := map[string]string{}
			for _, name := range c.ParamNames() {
				params[name] = c.Param(name)
			}
			args := []any{
				message,
				logs.Any("status", status),
				logs.Any("method", c.Request().Method),
				logs.Any("path", c.Path()),
				logs.Any("params", params),
				logs.Any("remote_ip", c.RealIP()),
				logs.Any("latency", finish.Sub(start)),
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
			status = resp.StatusCode()
			if status == 0 {
				status = http.StatusInternalServerError
			}
			return c.JSON(status, resp)
		}
		return err
	}
}

func (v *View) wrapSyncStores(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if v.getBoolSetting("handlers.allow_sync", c.Logger()).OrElse(true) {
			sync := strings.ToLower(c.Request().Header.Get("X-Solve-Sync"))
			c.Set(syncKey, sync == "1" || sync == "t" || sync == "true")
		} else {
			c.Set(syncKey, false)
		}
		return next(c)
	}
}

type authMethod func(c echo.Context) (bool, error)

func (v *View) extractAuth(authMethods ...authMethod) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			for _, method := range authMethods {
				ok, err := method(c)
				if err != nil {
					return err
				}
				if ok {
					return next(c)
				}
			}
			return errorResponse{
				Code:    http.StatusUnauthorized,
				Message: localize(c, "Unable to authorize."),
			}
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
	if err := syncStore(c, v.core.Sessions); err != nil {
		return false, err
	}
	session, err := v.core.Sessions.GetByCookie(cookie.Value)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if err := syncStore(c, v.core.Accounts); err != nil {
		return false, err
	}
	account, err := v.core.Accounts.Get(getContext(c), session.AccountID)
	if err != nil {
		return false, err
	}
	// Do not allow scope to login.
	if account.Kind == models.ScopeAccountKind {
		return false, nil
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
	if err := reusableBind(c, &form); err != nil {
		return false, nil
	}
	if form.Login == "" || form.Password == "" {
		return false, nil
	}
	if err := syncStore(c, v.core.Users); err != nil {
		return false, err
	}
	ctx := getContext(c)
	user, err := v.core.Users.GetByLogin(ctx, form.Login)
	if err != nil {
		if err == sql.ErrNoRows {
			resp := errorResponse{
				Code:    http.StatusUnauthorized,
				Message: localize(c, "User not found."),
			}
			return false, resp
		}
		return false, err
	}
	if !v.core.Users.CheckPassword(user, form.Password) {
		resp := errorResponse{
			Code:    http.StatusUnauthorized,
			Message: localize(c, "Invalid password."),
		}
		return false, resp
	}
	if err := syncStore(c, v.core.Accounts); err != nil {
		return false, err
	}
	account, err := v.core.Accounts.Get(ctx, user.ID)
	if err != nil {
		return false, err
	}
	if account.Kind != models.UserAccountKind {
		c.Logger().Errorf(
			"Account %v should have %v kind, but has %v",
			account.ID, models.UserAccountKind, account.Kind,
		)
		return false, fmt.Errorf("invalid account kind %q", account.Kind)
	}
	accountCtx, err := v.accounts.MakeContext(ctx, &account)
	if err != nil {
		return false, err
	}
	c.Set(accountCtxKey, accountCtx)
	c.Set(permissionCtxKey, accountCtx)
	return true, nil
}

type scopeUserAuthForm struct {
	ScopeID  int64  `json:"scope_id"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (v *View) scopeUserAuth(c echo.Context) (bool, error) {
	var form scopeUserAuthForm
	if err := reusableBind(c, &form); err != nil {
		return false, nil
	}
	if form.ScopeID == 0 || form.Login == "" || form.Password == "" {
		return false, nil
	}
	if err := syncStore(c, v.core.ScopeUsers); err != nil {
		return false, err
	}
	user, err := v.core.ScopeUsers.GetByScopeLogin(form.ScopeID, form.Login)
	if err != nil {
		if err == sql.ErrNoRows {
			resp := errorResponse{
				Code:    http.StatusUnauthorized,
				Message: localize(c, "User not found."),
			}
			return false, resp
		}
		return false, err
	}
	if !v.core.ScopeUsers.CheckPassword(user, form.Password) {
		resp := errorResponse{
			Code:    http.StatusUnauthorized,
			Message: localize(c, "Invalid password."),
		}
		return false, resp
	}
	if err := syncStore(c, v.core.Accounts); err != nil {
		return false, err
	}
	account, err := v.core.Accounts.Get(getContext(c), user.ID)
	if err != nil {
		return false, err
	}
	if account.Kind != models.ScopeUserAccountKind {
		c.Logger().Errorf(
			"Account %v should have %v kind, but has %v",
			account.ID, models.ScopeUserAccountKind, account.Kind,
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
				Code:    http.StatusForbidden,
				Message: localize(c, "Account missing permissions."),
			}
			ctx, ok := c.Get(permissionCtxKey).(perms.Permissions)
			if !ok {
				resp.MissingPermissions = names
				return resp
			}
			for _, name := range names {
				if !ctx.HasPermission(name) {
					resp.MissingPermissions = append(resp.MissingPermissions, name)
				}
			}
			if len(resp.MissingPermissions) > 0 {
				return resp
			}
			return next(c)
		}
	}
}

func (v *View) getBoolSetting(key string, logger echo.Logger) models.Option[bool] {
	value, err := v.core.Settings.GetBool(key)
	if err != nil {
		logger.Warn("Cannot get setting", logs.Any("key", key), err)
		return models.Empty[bool]()
	}
	return value
}

func (v *View) getInt64Setting(key string, logger echo.Logger) models.Option[int64] {
	value, err := v.core.Settings.GetInt64(key)
	if err != nil {
		logger.Warn("Cannot get setting", logs.Any("key", key), err)
		return models.Empty[int64]()
	}
	return value
}

type locale interface {
	Name() string
	Localize(text string, options ...func(*string)) string
	LocalizeKey(key string, text string, options ...func(*string)) string
	GetLocalizations() ([]Localization, error)
}

var stubLocaleValue stubLocale

func getLocale(c echo.Context) locale {
	locale, ok := c.Get(localeKey).(locale)
	if ok {
		return locale
	}
	return &stubLocaleValue
}

func localize(c echo.Context, text string, options ...func(*string)) string {
	return getLocale(c).Localize(text, options...)
}

func replaceField(name string, value any) func(*string) {
	return func(text *string) {
		*text = strings.ReplaceAll(*text, "{"+name+"}", fmt.Sprint(value))
	}
}

type stubLocale struct{}

func (stubLocale) Name() string {
	return ""
}

func (stubLocale) Localize(text string, options ...func(*string)) string {
	for _, option := range options {
		option(&text)
	}
	return text
}

func (l stubLocale) LocalizeKey(key string, text string, options ...func(*string)) string {
	return l.Localize(text, options...)
}

func (stubLocale) GetLocalizations() ([]Localization, error) {
	return nil, nil
}

type settingLocale struct {
	name     string
	settings *models.SettingStore
}

func (l *settingLocale) Name() string {
	return l.name
}

func (l *settingLocale) Localize(text string, options ...func(*string)) string {
	return l.LocalizeKey(getLocalizationKey(text), text, options...)
}

func (l *settingLocale) LocalizeKey(key string, text string, options ...func(*string)) string {
	settingKey := strings.Builder{}
	settingKey.WriteString("localization.")
	settingKey.WriteString(l.name)
	settingKey.WriteRune('.')
	settingKey.WriteString(key)
	if localized, err := l.settings.GetByKey(settingKey.String()); err == nil {
		text = localized.Value
	}
	for _, option := range options {
		option(&text)
	}
	return text
}

func (l *settingLocale) GetLocalizations() ([]Localization, error) {
	settings, err := l.settings.All(context.TODO(), 0, 0)
	if err != nil {
		return nil, err
	}
	defer func() { _ = settings.Close() }()
	prefix := "localization." + l.name + "."
	var localizations []Localization
	for settings.Next() {
		setting := settings.Row()
		if strings.HasPrefix(setting.Key, prefix) {
			localization := Localization{
				Key:  setting.Key[len(prefix):],
				Text: setting.Value,
			}
			localizations = append(localizations, localization)
		}
	}
	if err := settings.Err(); err != nil {
		return nil, err
	}
	return localizations, nil
}

func getLocalizationKey(text string) string {
	var key strings.Builder
	split := false
	for _, c := range text {
		if unicode.IsLetter(c) {
			if split {
				key.WriteRune('_')
				split = false
			}
			key.WriteRune(unicode.ToLower(c))
		} else {
			split = true
		}
	}
	return key.String()
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

func syncStore(c echo.Context, s any) error {
	store, ok := s.(models.CachedStore)
	if !ok {
		return nil
	}
	if sync, ok := c.Get(syncKey).(bool); ok && sync {
		return store.Sync(c.Request().Context())
	}
	return nil
}

func getPtr[T any](object T) *T {
	return &object
}

var (
	sqlRepeatableRead = gosql.WithIsolation(sql.LevelRepeatableRead)
	//lint:ignore U1000 Will be used in future.
	sqlReadOnly = gosql.WithReadOnly(true)
)

func sortFunc[T any](a []T, less func(T, T) bool) {
	impl := sortFuncImpl[T]{data: a, less: less}
	sort.Sort(&impl)
}

type sortFuncImpl[T any] struct {
	data []T
	less func(T, T) bool
}

func (s *sortFuncImpl[T]) Len() int {
	return len(s.data)
}

func (s *sortFuncImpl[T]) Swap(i, j int) {
	s.data[i], s.data[j] = s.data[j], s.data[i]
}

func (s *sortFuncImpl[T]) Less(i, j int) bool {
	return s.less(s.data[i], s.data[j])
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// reusableBind is required when we need to use Bind multiple times.
func reusableBind(c echo.Context, form any) error {
	if c.Request().Body != nil {
		body := c.Request().Body
		defer func() { _ = body.Close() }()
		buffer := bytes.Buffer{}
		c.Request().Body = io.NopCloser(io.TeeReader(body, &buffer))
		err := c.Bind(form)
		c.Request().Body = io.NopCloser(&buffer)
		return err
	}
	return c.Bind(form)
}

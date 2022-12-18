package api

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/labstack/echo/v4"
	"golang.org/x/text/language"

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
	visits   chan visitContext
}

func (v *View) StartDaemons() {
	v.visits = make(chan visitContext, 100)
	v.core.StartTask("visits", v.visitsDaemon)
}

type visitContext struct {
	Path   string
	Visit  models.Visit
	Logger echo.Logger
}

func (v *visitContext) Create(view *View) {
	if s := view.getBoolSetting(
		"log_visit."+v.Path, v.Logger,
	); s == nil || *s {
		func() {
			ctx, cancel := context.WithTimeout(
				context.Background(), 5*time.Second,
			)
			defer cancel()
			if err := view.core.Visits.Create(ctx, &v.Visit); err != nil {
				view.core.Logger().Error("Unable to create visit", err)
			}
		}()
	}
}

func (v *View) visitsDaemon(ctx context.Context) {
	for {
		select {
		case visit := <-v.visits:
			visit.Create(v)
		case <-ctx.Done():
			select {
			case visit := <-v.visits:
				visit.Create(v)
			default:
			}
			return
		}
	}
}

// Register registers handlers in specified group.
func (v *View) Register(g *echo.Group) {
	g.Use(wrapResponse, v.wrapSyncStores, v.logVisit, v.extractLocale)
	g.GET("/ping", v.ping)
	g.GET("/health", v.health)
	v.registerUserHandlers(g)
	v.registerRoleHandlers(g)
	v.registerSessionHandlers(g)
	v.registerContestHandlers(g)
	v.registerProblemHandlers(g)
	v.registerSolutionHandlers(g)
	v.registerCompilerHandlers(g)
	v.registerLocaleHandlers(g)
	v.registerFileHandlers(g)
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
	fileKey               = "file"
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

func (v *View) extractLocale(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		acceptLanguage := c.Request().Header.Get("Accept-Language")
		tags, _, err := language.ParseAcceptLanguage(acceptLanguage)
		if err != nil {
			return next(c)
		}
		for _, tag := range tags {
			if tag.String() != "en" && tag.String() != "ru" {
				continue
			}
			locale := settingLocale{
				name:     tag.String(),
				settings: v.core.Settings,
			}
			c.Set(localeKey, &locale)
			break
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
		if s := v.getBoolSetting("allow_sync", c.Logger()); s == nil || *s {
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
				Code:    http.StatusForbidden,
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
	if err := syncStore(c, v.core.Users); err != nil {
		return false, err
	}
	user, err := v.core.Users.GetByLogin(form.Login)
	if err != nil {
		if err == sql.ErrNoRows {
			resp := errorResponse{
				Code:    http.StatusForbidden,
				Message: localize(c, "User not found."),
			}
			return false, resp
		}
		return false, err
	}
	if !v.core.Users.CheckPassword(user, form.Password) {
		resp := errorResponse{
			Code:    http.StatusForbidden,
			Message: localize(c, "Invalid password."),
		}
		return false, resp
	}
	if err := syncStore(c, v.core.Accounts); err != nil {
		return false, err
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
				Code:    http.StatusForbidden,
				Message: localize(c, "Account missing permissions."),
			}
			ctx, ok := c.Get(permissionCtxKey).(managers.Permissions)
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

func (v *View) getStringSetting(key string, logger echo.Logger) *string {
	setting, err := v.core.Settings.GetByKey(key)
	if err != nil {
		if err != sql.ErrNoRows {
			logger.Error(
				"Unable to get setting",
				core.Any("key", key), err,
			)
		}
		return nil
	}
	if setting.Key != key {
		panic(fmt.Errorf("unexpected key %q != %q", setting.Key, key))
	}
	return &setting.Value
}

func (v *View) getBoolSetting(key string, logger echo.Logger) *bool {
	setting := v.getStringSetting(key, logger)
	if setting == nil {
		return nil
	}
	switch strings.ToLower(*setting) {
	case "1", "t", "true":
		return getPtr(true)
	case "0", "f", "false":
		return getPtr(false)
	default:
		logger.Warn(
			"Setting has invalid value",
			core.Any("key", key),
			core.Any("value", *setting),
		)
		return nil
	}
}

type locale interface {
	Name() string
	Localize(text string, options ...func(*string)) string
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
	key := l.getLocalizationKey(text)
	if localized, err := l.settings.GetByKey(key); err == nil {
		text = localized.Value
	}
	for _, option := range options {
		option(&text)
	}
	return text
}

func (l *settingLocale) GetLocalizations() ([]Localization, error) {
	settings, err := l.settings.All()
	if err != nil {
		return nil, err
	}
	prefix := "localization." + l.name + "."
	var localizations []Localization
	for _, setting := range settings {
		if strings.HasPrefix(setting.Key, prefix) {
			localization := Localization{
				Key:  setting.Key[len(prefix):],
				Text: setting.Value,
			}
			localizations = append(localizations, localization)
		}
	}
	return localizations, nil
}

func (l *settingLocale) getLocalizationKey(text string) string {
	var key strings.Builder
	key.WriteString("localization.")
	key.WriteString(l.name)
	key.WriteRune('.')
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

func syncStore(c echo.Context, s models.Store) error {
	if sync, ok := c.Get(syncKey).(bool); ok && sync {
		return s.Sync(c.Request().Context())
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

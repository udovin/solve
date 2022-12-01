package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/nsf/jsondiff"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/migrations"
)

type TestEnv struct {
	tb     testing.TB
	checks *testCheckState
	Core   *core.Core
	Server *httptest.Server
	Client *testClient
	Socket *testClient
	Now    time.Time
	Rand   *rand.Rand
}

func (e *TestEnv) SyncStores() {
	ctx := context.Background()
	if err := e.Core.Accounts.Sync(ctx); err != nil {
		e.tb.Fatal("Error:", err)
	}
	if err := e.Core.Users.Sync(ctx); err != nil {
		e.tb.Fatal("Error:", err)
	}
	if err := e.Core.Sessions.Sync(ctx); err != nil {
		e.tb.Fatal("Error:", err)
	}
	if err := e.Core.Roles.Sync(ctx); err != nil {
		e.tb.Fatal("Error:", err)
	}
	if err := e.Core.RoleEdges.Sync(ctx); err != nil {
		e.tb.Fatal("Error:", err)
	}
	if err := e.Core.AccountRoles.Sync(ctx); err != nil {
		e.tb.Fatal("Error:", err)
	}
	if err := e.Core.Contests.Sync(ctx); err != nil {
		e.tb.Fatal("Error:", err)
	}
	if err := e.Core.Problems.Sync(ctx); err != nil {
		e.tb.Fatal("Error:", err)
	}
	if err := e.Core.Compilers.Sync(ctx); err != nil {
		e.tb.Fatal("Error:", err)
	}
}

func (e *TestEnv) CreateUserRoles(login string, roles ...string) error {
	for _, role := range roles {
		if _, err := e.Socket.CreateUserRole(context.Background(), login, role); err != nil {
			return err
		}
	}
	return nil
}

func (e *TestEnv) Check(data any) {
	e.checks.Check(data)
}

func (e *TestEnv) Close() {
	e.Server.Close()
	e.Core.Stop()
	_ = db.ApplyMigrations(context.Background(), e.Core.DB, "solve", migrations.Schema, db.WithZeroMigration)
	_ = db.ApplyMigrations(context.Background(), e.Core.DB, "solve_data", migrations.Data, db.WithZeroMigration)
	e.checks.Close()
}

func NewTestEnv(tb testing.TB) *TestEnv {
	env := TestEnv{
		tb:     tb,
		checks: newTestCheckState(tb),
		Now:    time.Date(2020, 1, 1, 10, 0, 0, 0, time.UTC),
		Rand:   rand.New(rand.NewSource(42)),
	}
	cfg := config.Config{
		DB: config.DB{
			Options: config.SQLiteOptions{Path: ":memory:"},
		},
		Security: &config.Security{
			PasswordSalt: "qwerty123",
		},
		Storage: &config.Storage{
			Options: config.LocalStorageOptions{
				FilesDir: tb.TempDir(),
			},
		},
	}
	if _, ok := tb.(*testing.B); ok || os.Getenv("TEST_ENABLE_LOGS") != "1" {
		log.SetLevel(log.OFF)
		cfg.LogLevel = config.LogLevel(log.OFF)
	}
	if c, err := core.NewCore(cfg); err != nil {
		tb.Fatal("Error:", err)
	} else {
		env.Core = c
	}
	env.Core.SetupAllStores()
	ctx := context.Background()
	_ = db.ApplyMigrations(ctx, env.Core.DB, "solve", migrations.Schema, db.WithZeroMigration)
	_ = db.ApplyMigrations(ctx, env.Core.DB, "solve_data", migrations.Data, db.WithZeroMigration)
	if err := db.ApplyMigrations(ctx, env.Core.DB, "solve", migrations.Schema); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := db.ApplyMigrations(ctx, env.Core.DB, "solve_data", migrations.Data); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := env.Core.Start(); err != nil {
		tb.Fatal("Error:", err)
	}
	e := echo.New()
	e.Logger = env.Core.Logger()
	view := NewView(env.Core)
	nowFn := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(nowKey, env.Now)
			return next(c)
		}
	}
	e.Use(nowFn)
	view.Register(e.Group("/api"))
	view.RegisterSocket(e.Group("/socket"))
	env.Server = httptest.NewServer(e)
	env.Client = newTestClient(env.Server.URL + "/api")
	env.Socket = newTestClient(env.Server.URL + "/socket")
	return &env
}

type TestUser struct {
	User
	Password string
	env      *TestEnv
}

func (u *TestUser) LoginClient() {
	_, err := u.env.Client.Login(context.Background(), u.User.Login, u.Password)
	if err != nil {
		u.env.tb.Fatal("Error:", err)
	}
}

func (u *TestUser) LogoutClient() {
	if err := u.env.Client.Logout(context.Background()); err != nil {
		u.env.tb.Fatal("Error:", err)
	}
}

func (u *TestUser) AddRoles(names ...string) {
	if err := u.env.CreateUserRoles(u.User.Login, names...); err != nil {
		u.env.tb.Fatal("Error:", err)
	}
	u.env.SyncStores()
}

func NewTestUser(e *TestEnv) *TestUser {
	login := fmt.Sprintf("login-%d", e.Rand.Int31())
	password := fmt.Sprintf("password-%d", e.Rand.Int63())
	user, err := e.Client.Register(context.Background(), RegisterUserForm{
		Login:      login,
		Email:      login + "@example.com",
		Password:   password,
		FirstName:  "First",
		LastName:   "Last",
		MiddleName: "Middle",
	})
	if err != nil {
		e.tb.Fatal("Error:", err)
	}
	return &TestUser{
		User:     user,
		Password: password,
		env:      e,
	}
}

type testCheckState struct {
	tb     testing.TB
	checks []json.RawMessage
	pos    int
	reset  bool
	path   string
}

func (s *testCheckState) Check(data any) {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		s.tb.Fatal("Unable to marshal data:", data)
	}
	if s.pos > len(s.checks) {
		s.tb.Fatalf("Invalid check position: %d", s.pos)
	}
	if s.pos == len(s.checks) {
		if s.reset {
			s.checks = append(s.checks, raw)
			s.pos++
			return
		}
		s.tb.Errorf("Unexpected check with data: %s", raw)
		s.tb.Fatalf("Maybe you should use: TEST_RESET_DATA=1")
	}
	options := jsondiff.DefaultConsoleOptions()
	diff, report := jsondiff.Compare(s.checks[s.pos], raw, &options)
	if diff != jsondiff.FullMatch {
		if s.reset {
			s.checks[s.pos] = raw
			s.pos++
			return
		}
		s.tb.Errorf("Unexpected result difference: %s", report)
		s.tb.Fatalf("Maybe you should use: TEST_RESET_DATA=1")
	}
	s.pos++
}

func (s *testCheckState) Close() {
	if s.reset {
		if s.pos == 0 {
			_ = os.Remove(s.path)
			return
		}
		raw, err := json.MarshalIndent(s.checks, "", "  ")
		if err != nil {
			s.tb.Fatal("Unable to marshal test data:", err)
		}
		if err := os.WriteFile(
			s.path, raw, os.ModePerm,
		); err != nil {
			s.tb.Fatal("Error:", err)
		}
	}
}

func newTestCheckState(tb testing.TB) *testCheckState {
	state := testCheckState{
		tb:    tb,
		reset: os.Getenv("TEST_RESET_DATA") == "1",
		path:  filepath.Join("testdata", tb.Name()+".json"),
	}
	if !state.reset {
		file, err := os.Open(state.path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				tb.Fatal("Error:", err)
			}
		} else {
			defer file.Close()
			if err := json.NewDecoder(file).Decode(&state.checks); err != nil {
				tb.Fatal("Error:", err)
			}
		}
	}
	return &state
}

type testClient struct {
	*Client
}

type testJar struct {
	mutex   sync.Mutex
	cookies map[string]*http.Cookie
}

func (j *testJar) Cookies(*url.URL) []*http.Cookie {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	var cookies []*http.Cookie
	for _, cookie := range j.cookies {
		cookies = append(cookies, cookie)
	}
	return cookies
}

func (j *testJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	if j.cookies == nil {
		j.cookies = map[string]*http.Cookie{}
	}
	for _, cookie := range cookies {
		j.cookies[cookie.Name] = cookie
	}
}

func newTestClient(endpoint string) *testClient {
	client := NewClient(endpoint)
	client.client.Jar = &testJar{}
	client.Headers = map[string]string{"X-Solve-Sync": "1"}
	return &testClient{client}
}

func (c *testClient) Status() (Status, error) {
	req, err := http.NewRequest(http.MethodGet, c.getURL("/v0/status"), nil)
	if err != nil {
		return Status{}, err
	}
	var respData Status
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *testClient) ObserveUser(login string) (User, error) {
	req, err := http.NewRequest(
		http.MethodGet, c.getURL("/v0/users/%s", login), nil,
	)
	if err != nil {
		return User{}, err
	}
	var respData User
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *testClient) ObserveContests() (Contests, error) {
	req, err := http.NewRequest(
		http.MethodGet, c.getURL("/v0/contests"), nil,
	)
	if err != nil {
		return Contests{}, err
	}
	var respData Contests
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *testClient) CreateContest(form createContestForm) (Contest, error) {
	data, err := json.Marshal(form)
	if err != nil {
		return Contest{}, err
	}
	req, err := http.NewRequest(
		http.MethodPost, c.getURL("/v0/contests"),
		bytes.NewReader(data),
	)
	if err != nil {
		return Contest{}, err
	}
	var respData Contest
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *testClient) CreateContestProblem(
	contestID int64,
	form createContestProblemForm,
) (ContestProblem, error) {
	data, err := json.Marshal(form)
	if err != nil {
		return ContestProblem{}, err
	}
	req, err := http.NewRequest(
		http.MethodPost,
		c.getURL("/v0/contests/%d/problems", contestID),
		bytes.NewReader(data),
	)
	if err != nil {
		return ContestProblem{}, err
	}
	var respData ContestProblem
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *testClient) CreateRoleRole(role string, child string) (Role, error) {
	req, err := http.NewRequest(
		http.MethodPost, c.getURL("/v0/roles/%s/roles/%s", role, child),
		nil,
	)
	if err != nil {
		return Role{}, err
	}
	var respData Role
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *testClient) DeleteRoleRole(role string, child string) (Role, error) {
	req, err := http.NewRequest(
		http.MethodDelete, c.getURL("/v0/roles/%s/roles/%s", role, child),
		nil,
	)
	if err != nil {
		return Role{}, err
	}
	var respData Role
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func TestPing(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	if err := e.Client.Ping(context.Background()); err != nil {
		t.Fatal("Error:", err)
	}
}

func TestHealth(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	if err := e.Client.Health(context.Background()); err != nil {
		t.Fatal("Error:", err)
	}
}

func TestHealthUnhealthy(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	if err := e.Core.DB.Close(); err != nil {
		t.Fatal("Error:", err)
	}
	err := e.Client.Health(context.Background())
	resp, ok := err.(statusCodeResponse)
	if !ok {
		t.Fatal("Invalid error:", err)
	}
	expectStatus(t, http.StatusInternalServerError, resp.StatusCode())
}

func expectStatus(tb testing.TB, expected, got int) {
	if got != expected {
		tb.Fatalf("Expected %v, got %v", expected, got)
	}
}

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	_ "github.com/udovin/solve/migrations"
)

type testCheckState struct {
	tb     testing.TB
	checks []json.RawMessage
	pos    int
	reset  bool
	path   string
}

func (s *testCheckState) Check(data any) {
	raw, err := json.MarshalIndent(data, "  ", "  ")
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
		s.tb.Fatalf("Unexpected check with data: %s", raw)
	}
	options := jsondiff.DefaultConsoleOptions()
	diff, report := jsondiff.Compare(s.checks[s.pos], raw, &options)
	if diff != jsondiff.FullMatch {
		if s.reset {
			s.checks[s.pos] = raw
			s.pos++
			return
		}
		s.tb.Error("Unexpected result difference:")
		s.tb.Fatalf(report)
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

var (
	testView   *View
	testEcho   *echo.Echo
	testSrv    *httptest.Server
	testChecks *testCheckState
	testAPI    *testClient
	testNow    = time.Date(2020, 1, 1, 10, 0, 0, 0, time.UTC)
)

func wrapTestNow(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set(nowKey, testNow)
		return next(c)
	}
}

func testSetup(tb testing.TB) {
	testChecks = newTestCheckState(tb)
	cfg := config.Config{
		DB: config.DB{
			Options: config.SQLiteOptions{Path: ":memory:"},
		},
		Security: &config.Security{
			PasswordSalt: "qwerty123",
		},
		Storage: &config.Storage{
			FilesDir: tb.TempDir(),
		},
	}
	if _, ok := tb.(*testing.B); ok {
		log.SetLevel(log.OFF)
		cfg.LogLevel = config.LogLevel(log.OFF)
	}
	c, err := core.NewCore(cfg)
	if err != nil {
		tb.Fatal("Error:", err)
	}
	c.SetupAllStores()
	if err := db.ApplyMigrations(context.Background(), c.DB, db.WithZeroMigration); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := db.ApplyMigrations(context.Background(), c.DB); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := core.CreateData(context.Background(), c); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := c.Start(); err != nil {
		tb.Fatal("Error:", err)
	}
	testEcho = echo.New()
	testEcho.Logger = c.Logger()
	testView = NewView(c)
	testEcho.Use(wrapTestNow)
	testView.Register(testEcho.Group("/api"))
	testView.RegisterSocket(testEcho.Group("/socket"))
	testSrv = httptest.NewServer(testEcho)
	testAPI = newTestClient(testSrv.URL + "/api")
}

func testTeardown(tb testing.TB) {
	testSrv.Close()
	testView.core.Stop()
	_ = db.ApplyMigrations(context.Background(), testView.core.DB, db.WithZeroMigration)
	testChecks.Close()
}

func testCheck(data any) {
	testChecks.Check(data)
}

type testClient struct {
	Client
}

type testJar struct {
	mutex   sync.Mutex
	cookies []*http.Cookie
}

func (j *testJar) Cookies(*url.URL) []*http.Cookie {
	return j.cookies
}

func (j *testJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	j.cookies = append(j.cookies, cookies...)
}

func newTestClient(endpoint string) *testClient {
	return &testClient{
		Client: Client{
			endpoint: endpoint,
			client: http.Client{
				Timeout: time.Second,
				Jar:     &testJar{},
			},
		},
	}
}

func (c *testClient) Register(form registerUserForm) (User, error) {
	data, err := json.Marshal(form)
	if err != nil {
		return User{}, err
	}
	req, err := http.NewRequest(
		http.MethodPost, c.getURL("/v0/register"),
		bytes.NewReader(data),
	)
	if err != nil {
		return User{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return User{}, err
	}
	if resp.StatusCode != http.StatusCreated {
		var respData errorResponse
		if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
			return User{}, err
		}
		return User{}, &respData
	}
	var respData User
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return User{}, err
	}
	return respData, nil
}

func (c *testClient) Logout() error {
	req, err := http.NewRequest(http.MethodPost, c.getURL("/v0/logout"), nil)
	if err != nil {
		return err
	}
	_, err = c.doRequest(req, http.StatusOK, nil)
	return err
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

func (c *testClient) ObserveContest(id int64) (Contest, error) {
	req, err := http.NewRequest(
		http.MethodGet, c.getURL("/v0/contests/%d", id), nil,
	)
	if err != nil {
		return Contest{}, err
	}
	var respData Contest
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

func (c *testClient) CreateRoleRole(role string, child string) (Roles, error) {
	req, err := http.NewRequest(
		http.MethodPost, c.getURL("/v0/roles/%s/roles/%s", role, child),
		nil,
	)
	if err != nil {
		return Roles{}, err
	}
	var respData Roles
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *testClient) DeleteRoleRole(role string, child string) (Roles, error) {
	req, err := http.NewRequest(
		http.MethodDelete, c.getURL("/v0/roles/%s/roles/%s", role, child),
		nil,
	)
	if err != nil {
		return Roles{}, err
	}
	var respData Roles
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *testClient) CreateUserRole(login string, role string) (Roles, error) {
	req, err := http.NewRequest(
		http.MethodPost, c.getURL("/v0/users/%s/roles/%s", login, role),
		nil,
	)
	if err != nil {
		return Roles{}, err
	}
	var respData Roles
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *testClient) DeleteUserRole(login string, role string) (Roles, error) {
	req, err := http.NewRequest(
		http.MethodDelete, c.getURL("/v0/users/%s/roles/%s", login, role),
		nil,
	)
	if err != nil {
		return Roles{}, err
	}
	var respData Roles
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func testSocketCreateUserRole(login string, role string) (Roles, error) {
	req := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/socket/v0/users/%s/roles/%s", login, role), nil,
	)
	var resp Roles
	err := doSocketRequest(req, http.StatusCreated, &resp)
	return resp, err
}

func testSocketCreateUserRoles(login string, roles ...string) error {
	for _, role := range roles {
		if _, err := testSocketCreateUserRole(login, role); err != nil {
			return err
		}
	}
	return nil
}

func doSocketRequest(req *http.Request, code int, resp any) error {
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		return err
	}
	if rec.Code != code {
		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			return err
		}
		return &resp
	}
	return json.NewDecoder(rec.Body).Decode(resp)
}

func testHandler(req *http.Request, rec *httptest.ResponseRecorder) error {
	c := testEcho.NewContext(req, rec)
	testEcho.Router().Find(req.Method, req.URL.Path, c)
	return c.Handler()(c)
}

func TestPing(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	if err := testAPI.Ping(context.Background()); err != nil {
		t.Fatal("Error:", err)
	}
}

func TestHealth(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	if err := testAPI.Health(context.Background()); err != nil {
		t.Fatal("Error:", err)
	}
}

func TestHealthUnhealthy(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	if err := testView.core.DB.Close(); err != nil {
		t.Fatal("Error:", err)
	}
	err := testAPI.Health(context.Background())
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

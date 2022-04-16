package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/migrations"
	"github.com/udovin/solve/models"
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
	if string(s.checks[s.pos]) != string(raw) {
		if s.reset {
			s.checks[s.pos] = raw
			s.pos++
			return
		}
		s.tb.Fatalf("Unexpected check with data: %s, expected: %s", raw, s.checks[s.pos])
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
)

func testSetup(tb testing.TB) {
	testChecks = newTestCheckState(tb)
	cfg := config.Config{
		DB: config.DB{
			Options: config.SQLiteOptions{Path: ":memory:"},
		},
		Security: &config.Security{
			PasswordSalt: "qwerty123",
		},
	}
	if _, ok := tb.(*testing.B); ok {
		cfg.LogLevel = config.LogLevel(log.OFF)
	}
	c, err := core.NewCore(cfg)
	if err != nil {
		tb.Fatal("Error:", err)
	}
	c.SetupAllStores()
	if err := migrations.Unapply(c, true); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := migrations.Apply(c); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := c.Start(); err != nil {
		tb.Fatal("Error:", err)
	}
	testEcho = echo.New()
	testView = NewView(c)
	testView.Register(testEcho.Group("/api"))
	testView.RegisterSocket(testEcho.Group("/socket"))
	testSrv = httptest.NewServer(testEcho)
	testAPI = newTestClient(testSrv.URL + "/api")
}

func testTeardown(tb testing.TB) {
	testSrv.Close()
	testView.core.Stop()
	_ = migrations.Unapply(testView.core, true)
	testChecks.Close()
}

func testCheck(data any) {
	testChecks.Check(data)
}

type testClient struct {
	Endpoint string
	cookies  []*http.Cookie
	client   http.Client
}

func newTestClient(endpoint string) *testClient {
	return &testClient{
		Endpoint: endpoint,
		client:   http.Client{Timeout: time.Second},
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

func (c *testClient) Login(login, password string) (Session, error) {
	data, err := json.Marshal(map[string]string{
		"login":    login,
		"password": password,
	})
	if err != nil {
		return Session{}, err
	}
	req, err := http.NewRequest(
		http.MethodPost, c.getURL("/v0/login"),
		bytes.NewReader(data),
	)
	if err != nil {
		return Session{}, err
	}
	var respData Session
	err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *testClient) Logout() error {
	req, err := http.NewRequest(http.MethodPost, c.getURL("/v0/logout"), nil)
	if err != nil {
		return err
	}
	return c.doRequest(req, http.StatusOK, nil)
}

func (c *testClient) Status() (Status, error) {
	req, err := http.NewRequest(http.MethodGet, c.getURL("/v0/status"), nil)
	if err != nil {
		return Status{}, err
	}
	var respData Status
	err = c.doRequest(req, http.StatusOK, &respData)
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
	err = c.doRequest(req, http.StatusOK, &respData)
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
	err = c.doRequest(req, http.StatusOK, &respData)
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
	err = c.doRequest(req, http.StatusOK, &respData)
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
	err = c.doRequest(req, http.StatusCreated, &respData)
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
	err = c.doRequest(req, http.StatusCreated, &respData)
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
	err = c.doRequest(req, http.StatusCreated, &respData)
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
	err = c.doRequest(req, http.StatusOK, &respData)
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
	err = c.doRequest(req, http.StatusCreated, &respData)
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
	err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *testClient) getURL(path string, args ...any) string {
	return c.Endpoint + fmt.Sprintf(path, args...)
}

func (c *testClient) doRequest(req *http.Request, code int, respData any) error {
	req.Header.Add("Content-Type", "application/json")
	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != code {
		var respData errorResponse
		if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
			return err
		}
		return &respData
	}
	c.cookies = append(c.cookies, resp.Cookies()...)
	if respData != nil {
		return json.NewDecoder(resp.Body).Decode(respData)
	}
	return nil
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
	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		t.Fatal("Error:", err)
	}
	expectStatus(t, http.StatusOK, rec.Code)
}

func TestHealth(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		t.Fatal("Error:", err)
	}
	expectStatus(t, http.StatusOK, rec.Code)
}

func TestHealthUnhealthy(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	if err := testView.core.DB.Close(); err != nil {
		t.Fatal("Error:", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		t.Fatal("Error:", err)
	}
	expectStatus(t, http.StatusInternalServerError, rec.Code)
}

func TestSessionAuth(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "123_qwerty123"})
	rec := httptest.NewRecorder()
	c := testEcho.NewContext(req, rec)
	handler := testView.sessionAuth(testView.ping)
	if err := handler(c); err != nil {
		t.Fatal("Error:", err)
	}
	expectStatus(t, http.StatusOK, rec.Code)
}

func TestUserAuth(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	data, err := json.Marshal(map[string]string{
		"login":    "test",
		"password": "qwerty123",
	})
	if err != nil {
		t.Fatal("Error:", err)
	}
	req := httptest.NewRequest(
		http.MethodPost, "/ping", bytes.NewReader(data),
	)
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := testEcho.NewContext(req, rec)
	handler := testView.userAuth(testView.ping)
	if err := handler(c); err != nil {
		t.Fatal("Error:", err)
	}
	expectStatus(t, http.StatusForbidden, rec.Code)
}

func TestRequireAuth(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	c := testEcho.NewContext(req, rec)
	handler := testView.requireAuth(testView.ping)
	if err := handler(c); err != nil {
		t.Fatal("Error:", err)
	}
	expectStatus(t, http.StatusForbidden, rec.Code)
}

func TestExtractAuthRoles(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	c := testEcho.NewContext(req, rec)
	handler := testView.extractAuthRoles(testView.ping)
	if err := handler(c); err != nil {
		t.Fatal("Error:", err)
	}
	expectStatus(t, http.StatusOK, rec.Code)
}

func TestRequireAuthRole(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	c := testEcho.NewContext(req, rec)
	handler := testView.requireAuthRole(models.ObserveUserRole)(testView.ping)
	if err := handler(c); err != nil {
		t.Fatal("Error:", err)
	}
	expectStatus(t, http.StatusOK, rec.Code)
}

func expectStatus(tb testing.TB, expected, got int) {
	if got != expected {
		tb.Fatalf("Expected %v, got %v", expected, got)
	}
}

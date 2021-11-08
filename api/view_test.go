package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/migrations"
	"github.com/udovin/solve/models"
)

var (
	testView *View
	testSrv  *echo.Echo
)

func testSetup(tb testing.TB) {
	cfg := config.Config{
		DB: config.DB{
			Options: config.SQLiteOptions{Path: ":memory:"},
		},
		Security: config.Security{
			PasswordSalt: config.Secret{
				Type: config.DataSecret,
				Data: "qwerty123",
			},
		},
	}
	c, err := core.NewCore(cfg)
	if err != nil {
		tb.Fatal("Error:", err)
	}
	if err := c.SetupAllStores(); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := migrations.Unapply(c); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := migrations.Apply(c); err != nil {
		tb.Fatal("Error:", err)
	}
	if err := c.Start(); err != nil {
		tb.Fatal("Error:", err)
	}
	testSrv = echo.New()
	testView = NewView(c)
	testView.Register(testSrv.Group("/api/v0"))
	testView.RegisterSocket(testSrv.Group("/socket/v0"))
}

func testTeardown(tb testing.TB) {
	_ = migrations.Unapply(testView.core)
	testView.core.Stop()
}

func testHandler(req *http.Request, rec *httptest.ResponseRecorder) error {
	c := testSrv.NewContext(req, rec)
	testSrv.Router().Find(req.Method, req.URL.Path, c)
	return c.Handler()(c)
}

type testClient struct {
	cookies []*http.Cookie
}

func newTestClient() *testClient {
	return &testClient{}
}

func (c *testClient) Register(form registerUserForm) (User, error) {
	data, err := json.Marshal(form)
	if err != nil {
		return User{}, err
	}
	req := httptest.NewRequest(
		http.MethodPost, "/api/v0/register", bytes.NewReader(data),
	)
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		return User{}, err
	}
	if rec.Code != http.StatusCreated {
		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			return User{}, err
		}
		return User{}, &resp
	}
	var resp User
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		return User{}, err
	}
	return resp, nil
}

func (c *testClient) Login(login, password string) (Session, error) {
	data, err := json.Marshal(map[string]string{
		"login":    login,
		"password": password,
	})
	if err != nil {
		return Session{}, err
	}
	req := httptest.NewRequest(
		http.MethodPost, "/api/v0/login", bytes.NewReader(data),
	)
	var resp Session
	err = c.doRequest(req, http.StatusCreated, &resp)
	return resp, err
}

func (c *testClient) Logout() error {
	req := httptest.NewRequest(http.MethodPost, "/api/v0/logout", nil)
	err := c.doRequest(req, http.StatusOK, nil)
	return err
}

func (c *testClient) Status() (Status, error) {
	req := httptest.NewRequest(http.MethodGet, "/api/v0/status", nil)
	var resp Status
	err := c.doRequest(req, http.StatusOK, &resp)
	return resp, err
}

func (c *testClient) ObserveUser(login string) (User, error) {
	req := httptest.NewRequest(
		http.MethodGet, fmt.Sprintf("/api/v0/users/%s", login), nil,
	)
	var resp User
	err := c.doRequest(req, http.StatusOK, &resp)
	return resp, err
}

func (c *testClient) ObserveContests() (Contests, error) {
	req := httptest.NewRequest(http.MethodGet, "/api/v0/contests", nil)
	var resp Contests
	err := c.doRequest(req, http.StatusOK, &resp)
	return resp, err
}

func (c *testClient) ObserveContest(id int) (Contests, error) {
	req := httptest.NewRequest(
		http.MethodGet, fmt.Sprintf("/api/v0/contests/%d", id), nil,
	)
	var resp Contests
	err := c.doRequest(req, http.StatusOK, &resp)
	return resp, err
}

func (c *testClient) CreateContest(form createContestForm) (Contest, error) {
	data, err := json.Marshal(form)
	if err != nil {
		return Contest{}, err
	}
	req := httptest.NewRequest(
		http.MethodPost, "/api/v0/contests", bytes.NewReader(data),
	)
	var resp Contest
	err = c.doRequest(req, http.StatusCreated, &resp)
	return resp, err
}

func (c *testClient) doRequest(req *http.Request, code int, resp interface{}) error {
	req.Header.Add("Content-Type", "application/json")
	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}
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
	c.cookies = append(c.cookies, rec.Result().Cookies()...)
	if resp != nil {
		return json.NewDecoder(rec.Body).Decode(resp)
	}
	return nil
}

func testSocketCreateUserRole(login string, role string) ([]Role, error) {
	req := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/socket/v0/users/%s/roles/%s", login, role), nil,
	)
	var resp []Role
	err := doSocketRequest(req, http.StatusCreated, &resp)
	return resp, err
}

func doSocketRequest(req *http.Request, code int, resp interface{}) error {
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

func TestPing(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v0/ping", nil)
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		t.Fatal("Error:", err)
	}
	expectStatus(t, http.StatusOK, rec.Code)
}

func TestHealth(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v0/health", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/api/v0/health", nil)
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
	c := testSrv.NewContext(req, rec)
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
	c := testSrv.NewContext(req, rec)
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
	c := testSrv.NewContext(req, rec)
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
	c := testSrv.NewContext(req, rec)
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
	c := testSrv.NewContext(req, rec)
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

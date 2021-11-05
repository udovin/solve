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
			Driver:  config.SQLiteDriver,
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

func (c *testClient) setCookie(req *http.Request) {
	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}
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
	c.setCookie(req)
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		return Session{}, err
	}
	if rec.Code != http.StatusCreated {
		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			return Session{}, err
		}
		return Session{}, &resp
	}
	var resp Session
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		return Session{}, err
	}
	c.cookies = rec.Result().Cookies()
	return resp, nil
}

func (c *testClient) Logout() error {
	req := httptest.NewRequest(http.MethodPost, "/api/v0/logout", nil)
	c.setCookie(req)
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		return err
	}
	if rec.Code != http.StatusOK {
		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			return err
		}
		return &resp
	}
	c.cookies = rec.Result().Cookies()
	return nil
}

func (c *testClient) Status() (Status, error) {
	req := httptest.NewRequest(http.MethodGet, "/api/v0/status", nil)
	c.setCookie(req)
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		return Status{}, err
	}
	if rec.Code != http.StatusOK {
		var resp errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			return Status{}, err
		}
		return Status{}, &resp
	}
	var resp Status
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		return Status{}, err
	}
	return resp, nil
}

func (c *testClient) ObserveUser(login string) (User, error) {
	req := httptest.NewRequest(
		http.MethodGet, fmt.Sprintf("/api/v0/users/%s", login), nil,
	)
	c.setCookie(req)
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		return User{}, err
	}
	if rec.Code != http.StatusOK {
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

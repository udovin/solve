package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/core"
	"github.com/udovin/solve/migrations"
)

var (
	testView *View
	testSrv  *echo.Echo
)

func testSetup(tb testing.TB) {
	cfg := config.Config{
		DB: config.DB{
			Driver:  config.SQLiteDriver,
			Options: config.SQLiteOptions{Path: "?mode=memory"},
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
}

func testTeardown(tb testing.TB) {
	testView.core.Stop()
}

func TestPing(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	c := testSrv.NewContext(req, rec)
	if err := testView.ping(c); err != nil {
		t.Fatal("Error:", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected %v, got %v", http.StatusOK, rec.Code)
	}
}

func TestHealth(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := testSrv.NewContext(req, rec)
	if err := testView.health(c); err != nil {
		t.Fatal("Error:", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected %v, got %v", http.StatusOK, rec.Code)
	}
}

func TestHealthUnhealthy(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	if err := testView.core.DB.Close(); err != nil {
		t.Fatal("Error:", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := testSrv.NewContext(req, rec)
	if err := testView.health(c); err != nil {
		t.Fatal("Error:", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"Expected %v, got %v",
			http.StatusInternalServerError, rec.Code,
		)
	}
}

func TestLogVisit(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	c := testSrv.NewContext(req, rec)
	handler := testView.logVisit(testView.ping)
	if err := handler(c); err != nil {
		t.Fatal("Error:", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected %v, got %v", http.StatusOK, rec.Code)
	}
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
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected %v, got %v", http.StatusOK, rec.Code)
	}
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
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Expected %v, got %v", http.StatusForbidden, rec.Code)
	}
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
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Expected %v, got %v", http.StatusForbidden, rec.Code)
	}
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
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected %v, got %v", http.StatusOK, rec.Code)
	}
}

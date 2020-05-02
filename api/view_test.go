package api

import (
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
	if err := c.SetupAllManagers(); err != nil {
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
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := testSrv.NewContext(req, rec)
	if err := testView.ping(c); err != nil {
		t.Fatal("Error:", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected %v, got %v", http.StatusOK, rec.Code)
	}
}

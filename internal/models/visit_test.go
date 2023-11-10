package models

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestVisit(t *testing.T) {
	var e Visit
	if v := e.EventID(); v != 0 {
		t.Fatalf("Expected 0, got: %d", v)
	}
	epoch := time.Unix(0, 0)
	if v := e.EventTime(); !v.Equal(epoch) {
		t.Fatalf("Expected zero, got: %v", v)
	}
	e.ID = 100
	if v := e.EventID(); v != 100 {
		t.Fatalf("Expected %d, got: %d", 100, v)
	}
}

func TestVisitStore(t *testing.T) {
	store := NewVisitStore(testDB, "visit")
	req := httptest.NewRequest(http.MethodGet, "/text?q=123", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	ctx := echo.New().NewContext(req, httptest.NewRecorder())
	ctx.SetPath("/:test")
	visit := store.MakeFromContext(ctx)
	if visit.Path != "/text?q=123" {
		t.Fatalf("Expected %q, got: %q", "/text?q=123", visit.Path)
	}
}

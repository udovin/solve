package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserLoginScenario(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	registerUser(t, "test", "qwerty123")
	if err := testView.core.WithTx(context.Background(), func(tx *sql.Tx) error {
		if err := testView.core.Accounts.SyncTx(tx); err != nil {
			return err
		}
		if err := testView.core.Users.SyncTx(tx); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatal("Error:", err)
	}
	loginUser(t, "test", "qwerty123")
}

func registerUser(tb testing.TB, login, password string) {
	data, err := json.Marshal(map[string]string{
		"login":       login,
		"password":    password,
		"email":       "test@example.com",
		"first_name":  "First",
		"last_name":   "Last",
		"middle_name": "Middle",
	})
	if err != nil {
		tb.Fatal("Error:", err)
	}
	req := httptest.NewRequest(
		http.MethodPost, "/register", bytes.NewReader(data),
	)
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := testSrv.NewContext(req, rec)
	if err := testView.registerUser(c); err != nil {
		tb.Fatal("Error:", err)
	}
	expectStatus(tb, http.StatusCreated, rec.Code)
}

func loginUser(tb testing.TB, login, password string) {
	data, err := json.Marshal(map[string]string{
		"login":    login,
		"password": password,
	})
	if err != nil {
		tb.Fatal("Error:", err)
	}
	req := httptest.NewRequest(
		http.MethodPost, "/login", bytes.NewReader(data),
	)
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := testSrv.NewContext(req, rec)
	handler := testView.userAuth(testView.loginAccount)
	if err := handler(c); err != nil {
		tb.Fatal("Error:", err)
	}
	expectStatus(tb, http.StatusCreated, rec.Code)
}

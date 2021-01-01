package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/udovin/solve/models"
)

func TestUserLoginScenario(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	registerUser(t, "test", "qwerty123")
	syncManagers(t)
	observeUser(t, "test")
	loginUser(t, "test", "qwerty123")
}

func syncManagers(tb testing.TB) {
	if err := testView.core.WithTx(
		context.Background(),
		func(tx *sql.Tx) error {
			if err := testView.core.Accounts.SyncTx(tx); err != nil {
				return err
			}
			if err := testView.core.Users.SyncTx(tx); err != nil {
				return err
			}
			if err := testView.core.UserFields.SyncTx(tx); err != nil {
				return err
			}
			return nil
		},
	); err != nil {
		tb.Fatal("Error:", err)
	}
}

func registerUser(tb testing.TB, login, password string) User {
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
	var resp User
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		tb.Fatal("Error:", err)
	}
	return resp
}

func loginUser(tb testing.TB, login, password string) Session {
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
	var resp Session
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		tb.Fatal("Error:", err)
	}
	return resp
}

func observeUser(tb testing.TB, login string) {
	req := httptest.NewRequest(
		http.MethodGet, fmt.Sprintf("/users/%s", login), nil,
	)
	rec := httptest.NewRecorder()
	c := testSrv.NewContext(req, rec)
	c.SetParamNames("user")
	c.SetParamValues(login)
	handler := testView.sessionAuth(
		testView.requireAuthRole(models.ObserveUserRole)(
			testView.extractUser(
				testView.extractUserRoles(testView.observeUser),
			),
		),
	)
	if err := handler(c); err != nil {
		tb.Fatal("Error:", err)
	}
	expectStatus(tb, http.StatusOK, rec.Code)
}

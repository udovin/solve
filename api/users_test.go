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
)

func TestUserLoginScenario(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	testRegisterUser(t, "test", "qwerty123")
	testSyncManagers(t)
	testGetUser(t, "test")
	testGetUserRoles(t, "test")
	testLoginUser(t, "test", "qwerty123")
}

func testSyncManagers(tb testing.TB) {
	if err := testView.core.WithTx(
		context.Background(),
		func(tx *sql.Tx) error {
			if err := testView.core.Accounts.SyncTx(tx); err != nil {
				return err
			}
			if err := testView.core.Users.SyncTx(tx); err != nil {
				return err
			}
			if err := testView.core.Roles.SyncTx(tx); err != nil {
				return err
			}
			if err := testView.core.AccountRoles.SyncTx(tx); err != nil {
				return err
			}
			return nil
		},
	); err != nil {
		tb.Fatal("Error:", err)
	}
}

func testRegisterUser(tb testing.TB, login, password string) User {
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
		http.MethodPost, "/api/v0/register", bytes.NewReader(data),
	)
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		tb.Fatal("Error:", err)
	}
	expectStatus(tb, http.StatusCreated, rec.Code)
	var resp User
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		tb.Fatal("Error:", err)
	}
	return resp
}

func testLoginUser(tb testing.TB, login, password string) Session {
	data, err := json.Marshal(map[string]string{
		"login":    login,
		"password": password,
	})
	if err != nil {
		tb.Fatal("Error:", err)
	}
	req := httptest.NewRequest(
		http.MethodPost, "/api/v0/login", bytes.NewReader(data),
	)
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		tb.Fatal("Error:", err)
	}
	expectStatus(tb, http.StatusCreated, rec.Code)
	var resp Session
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		tb.Fatal("Error:", err)
	}
	return resp
}

func testGetUser(tb testing.TB, login string) User {
	req := httptest.NewRequest(
		http.MethodGet, fmt.Sprintf("/socket/v0/users/%s", login), nil,
	)
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		tb.Fatal("Error:", err)
	}
	expectStatus(tb, http.StatusOK, rec.Code)
	var resp User
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		tb.Fatal("Error:", err)
	}
	return resp
}

func testGetUserRoles(tb testing.TB, login string) []Role {
	req := httptest.NewRequest(
		http.MethodGet, fmt.Sprintf("/socket/v0/users/%s/roles", login), nil,
	)
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		tb.Fatal("Error:", err)
	}
	expectStatus(tb, http.StatusOK, rec.Code)
	var resp []Role
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		tb.Fatal("Error:", err)
	}
	return resp
}

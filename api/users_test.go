package api

import (
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
	client := newTestClient()
	if _, err := client.Register("test", "qwerty123"); err != nil {
		t.Fatal("Error:", err)
	}
	testSyncManagers(t)
	testSocketObserveUser(t, "test")
	testSocketObserveUserRoles(t, "test")
	if _, err := client.Login("test", "qwerty123"); err != nil {
		t.Fatal("Error:", err)
	}
	if status, err := client.Status(); err != nil {
		t.Fatal("Error:", err)
	} else {
		if status.User == nil {
			t.Fatal("Status should have user")
		}
		if status.Session == nil {
			t.Fatal("Status should have session")
		}
	}
	if err := client.Logout(); err != nil {
		t.Fatal("Error:", err)
	}
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

func testSocketObserveUser(tb testing.TB, login string) User {
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

func testSocketObserveUserRoles(tb testing.TB, login string) []Role {
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

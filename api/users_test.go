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

var testSimpleUser = registerUserForm{
	Login:      "test",
	Password:   "qwerty123",
	FirstName:  "First",
	LastName:   "Last",
	MiddleName: "Middle",
	Email:      "text@example.com",
}

func TestUserSimpleScenario(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	client := newTestClient()
	if _, err := client.Register(testSimpleUser); err != nil {
		t.Fatal("Error:", err)
	}
	testSyncManagers(t)
	if user, err := client.ObserveUser("test"); err != nil {
		t.Fatal("Error:", err)
	} else {
		testCheck(user)
	}
	testSocketObserveUserRoles(t, "test")
	if _, err := client.Login("test", "qwerty123"); err != nil {
		t.Fatal("Error:", err)
	}
	if status, err := client.Status(); err != nil {
		t.Fatal("Error:", err)
	} else {
		// Canonical tests does not support current timestamps.
		status.Session.CreateTime = 0
		status.Session.ExpireTime = 0
		testCheck(status)
	}
	if user, err := client.ObserveUser("test"); err != nil {
		t.Fatal("Error:", err)
	} else {
		testCheck(user)
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
			if err := testView.core.Contests.SyncTx(tx); err != nil {
				return err
			}
			if err := testView.core.Problems.SyncTx(tx); err != nil {
				return err
			}
			return nil
		},
	); err != nil {
		tb.Fatal("Error:", err)
	}
}

func testSocketObserveUserRoles(tb testing.TB, login string) Roles {
	req := httptest.NewRequest(
		http.MethodGet, fmt.Sprintf("/socket/v0/users/%s/roles", login), nil,
	)
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		tb.Fatal("Error:", err)
	}
	expectStatus(tb, http.StatusOK, rec.Code)
	var resp Roles
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		tb.Fatal("Error:", err)
	}
	return resp
}

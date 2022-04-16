package api

import (
	"context"
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
	if _, err := testAPI.Register(testSimpleUser); err != nil {
		t.Fatal("Error:", err)
	}
	testSyncManagers(t)
	if user, err := testAPI.ObserveUser("test"); err != nil {
		t.Fatal("Error:", err)
	} else {
		testCheck(user)
	}
	testSocketObserveUserRoles(t, "test")
	if _, err := testAPI.Login("test", "qwerty123"); err != nil {
		t.Fatal("Error:", err)
	}
	if status, err := testAPI.Status(); err != nil {
		t.Fatal("Error:", err)
	} else {
		// Canonical tests does not support current timestamps.
		status.Session.CreateTime = 0
		status.Session.ExpireTime = 0
		testCheck(status)
	}
	if user, err := testAPI.ObserveUser("test"); err != nil {
		t.Fatal("Error:", err)
	} else {
		testCheck(user)
	}
	if err := testAPI.Logout(); err != nil {
		t.Fatal("Error:", err)
	}
}

func testSyncManagers(tb testing.TB) {
	if err := testView.core.WrapTx(
		context.Background(),
		func(ctx context.Context) error {
			if err := testView.core.Accounts.Sync(ctx); err != nil {
				return err
			}
			if err := testView.core.Users.Sync(ctx); err != nil {
				return err
			}
			if err := testView.core.Roles.Sync(ctx); err != nil {
				return err
			}
			if err := testView.core.RoleEdges.Sync(ctx); err != nil {
				return err
			}
			if err := testView.core.AccountRoles.Sync(ctx); err != nil {
				return err
			}
			if err := testView.core.Contests.Sync(ctx); err != nil {
				return err
			}
			if err := testView.core.Problems.Sync(ctx); err != nil {
				return err
			}
			return nil
		},
		sqlReadOnly,
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

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestObserveRoles(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	req := httptest.NewRequest(http.MethodGet, "/roles", nil)
	rec := httptest.NewRecorder()
	c := testEcho.NewContext(req, rec)
	if err := testView.observeRoles(c); err != nil {
		t.Fatal("Error:", err)
	}
	expectStatus(t, http.StatusOK, rec.Code)
}

func TestCreateDeleteRole(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	created := createRole(t, "test_role")
	testCheck(created)
	testSyncManagers(t)
	deleted := deleteRole(t, created.ID)
	if created != deleted {
		t.Fatal("Invalid deleted role:", deleted)
	}
}

func TestRoleSimpleScenario(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	role1 := createRole(t, "role1")
	testCheck(role1)
	role2 := createRole(t, "role2")
	testCheck(role2)
	if _, err := testAPI.Register(testSimpleUser); err != nil {
		t.Fatal("Error:", err)
	}
	testSyncManagers(t)
	if _, err := testAPI.Login("test", "qwerty123"); err != nil {
		t.Fatal("Error:", err)
	}
	testSocketCreateUserRole("test", "admin_group")
	testSyncManagers(t)
	{
		roles, err := testAPI.CreateRoleRole("role1", "role2")
		if err != nil {
			t.Fatal("Error:", err)
		}
		testCheck(roles)
		testSyncManagers(t)
	}
	{
		roles, err := testAPI.DeleteRoleRole("role1", "role2")
		if err != nil {
			t.Fatal("Error:", err)
		}
		testCheck(roles)
	}
	{
		roles, err := testAPI.CreateUserRole("test", "role2")
		if err != nil {
			t.Fatal("Error:", err)
		}
		testCheck(roles)
		testSyncManagers(t)
	}
	{
		roles, err := testAPI.DeleteUserRole("test", "role2")
		if err != nil {
			t.Fatal("Error:", err)
		}
		testCheck(roles)
	}
}

func createRole(tb testing.TB, name string) Role {
	data, err := json.Marshal(map[string]string{
		"name": name,
	})
	if err != nil {
		tb.Fatal("Error:", err)
	}
	req := httptest.NewRequest(
		http.MethodPost, "/roles", bytes.NewReader(data),
	)
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := testEcho.NewContext(req, rec)
	if err := testView.createRole(c); err != nil {
		tb.Fatal("Error:", err)
	}
	expectStatus(tb, http.StatusCreated, rec.Code)
	var resp Role
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		tb.Fatal("Error:", err)
	}
	return resp
}

func deleteRole(tb testing.TB, role int64) Role {
	req := httptest.NewRequest(
		http.MethodDelete, fmt.Sprintf("/roles/%d", role), nil,
	)
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := testEcho.NewContext(req, rec)
	c.SetParamNames("role")
	c.SetParamValues(fmt.Sprint(role))
	handler := testView.extractRole(testView.deleteRole)
	if err := handler(c); err != nil {
		tb.Fatal("Error:", err)
	}
	expectStatus(tb, http.StatusOK, rec.Code)
	var resp Role
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		tb.Fatal("Error:", err)
	}
	return resp
}

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
	c := testSrv.NewContext(req, rec)
	if err := testView.observeRoles(c); err != nil {
		t.Fatal("Error:", err)
	}
	expectStatus(t, http.StatusOK, rec.Code)
}

func TestCreateDeleteRole(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	created := createRole(t, "test_role")
	if created.ID == 0 {
		t.Fatal("Invalid ID of role", created)
	}
	if created.Code != "test_role" {
		t.Fatal("Invalid code of role:", created)
	}
	testSyncManagers(t)
	deleted := deleteRole(t, created.ID)
	if created != deleted {
		t.Fatal("Invalid deleted role:", deleted)
	}
}

func createRole(tb testing.TB, code string) Role {
	data, err := json.Marshal(map[string]string{
		"code": code,
	})
	if err != nil {
		tb.Fatal("Error:", err)
	}
	req := httptest.NewRequest(
		http.MethodPost, "/roles", bytes.NewReader(data),
	)
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := testSrv.NewContext(req, rec)
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
	c := testSrv.NewContext(req, rec)
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

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
)

type testClient struct {
	cookies []*http.Cookie
}

func newTestClient() *testClient {
	return &testClient{}
}

func (c *testClient) setCookie(req *http.Request) {
	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}
}

func (c *testClient) Register(form registerUserForm) (User, error) {
	data, err := json.Marshal(form)
	if err != nil {
		return User{}, err
	}
	req := httptest.NewRequest(
		http.MethodPost, "/api/v0/register", bytes.NewReader(data),
	)
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		return User{}, err
	}
	if rec.Code != http.StatusCreated {
		var resp errorResp
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			return User{}, err
		}
		return User{}, &resp
	}
	var resp User
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		return User{}, err
	}
	return resp, nil
}

func (c *testClient) Login(login, password string) (Session, error) {
	data, err := json.Marshal(map[string]string{
		"login":    login,
		"password": password,
	})
	if err != nil {
		return Session{}, err
	}
	req := httptest.NewRequest(
		http.MethodPost, "/api/v0/login", bytes.NewReader(data),
	)
	c.setCookie(req)
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		return Session{}, err
	}
	if rec.Code != http.StatusCreated {
		var resp errorResp
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			return Session{}, err
		}
		return Session{}, &resp
	}
	var resp Session
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		return Session{}, err
	}
	c.cookies = rec.Result().Cookies()
	return resp, nil
}

func (c *testClient) Logout() error {
	req := httptest.NewRequest(http.MethodPost, "/api/v0/logout", nil)
	c.setCookie(req)
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		return err
	}
	if rec.Code != http.StatusOK {
		var resp errorResp
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			return err
		}
		return &resp
	}
	c.cookies = rec.Result().Cookies()
	return nil
}

func (c *testClient) Status() (Status, error) {
	req := httptest.NewRequest(http.MethodGet, "/api/v0/status", nil)
	c.setCookie(req)
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		return Status{}, err
	}
	if rec.Code != http.StatusOK {
		var resp errorResp
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			return Status{}, err
		}
		return Status{}, &resp
	}
	var resp Status
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		return Status{}, err
	}
	return resp, nil
}

func (c *testClient) ObserveUser(login string) (User, error) {
	req := httptest.NewRequest(
		http.MethodGet, fmt.Sprintf("/api/v0/users/%s", login), nil,
	)
	c.setCookie(req)
	rec := httptest.NewRecorder()
	if err := testHandler(req, rec); err != nil {
		return User{}, err
	}
	if rec.Code != http.StatusOK {
		var resp errorResp
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			return User{}, err
		}
		return User{}, &resp
	}
	var resp User
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		return User{}, err
	}
	return resp, nil
}

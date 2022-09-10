package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	endpoint string
	client   http.Client
}

// NewClient returns new API client.
func NewClient(endpoint string) *Client {
	c := Client{
		endpoint: endpoint,
		client: http.Client{
			Timeout: 5 * time.Second,
		},
	}
	return &c
}

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.getURL("/ping"), nil,
	)
	if err != nil {
		return err
	}
	_, err = c.doRequest(req, http.StatusOK, nil)
	return err
}

func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.getURL("/health"), nil,
	)
	if err != nil {
		return err
	}
	_, err = c.doRequest(req, http.StatusOK, nil)
	return err
}

func (c *Client) Login(ctx context.Context, login, password string) (Session, error) {
	data, err := json.Marshal(userAuthForm{
		Login:    login,
		Password: password,
	})
	if err != nil {
		return Session{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.getURL("/v0/login"), bytes.NewReader(data),
	)
	if err != nil {
		return Session{}, err
	}
	var respData Session
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *Client) getURL(path string, args ...any) string {
	return c.endpoint + fmt.Sprintf(path, args...)
}

func (c *Client) doRequest(req *http.Request, code int, respData any) (*http.Response, error) {
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != code {
		var respData errorResponse
		if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
			return nil, errorWithCode{
				Err:  err,
				Code: resp.StatusCode,
			}
		}
		respData.Code = resp.StatusCode
		return nil, &respData
	}
	if respData != nil {
		return nil, json.NewDecoder(resp.Body).Decode(respData)
	}
	return resp, nil
}

type errorWithCode struct {
	Err  error
	Code int
}

func (r errorWithCode) Error() string {
	return r.Err.Error()
}

func (r errorWithCode) Unwrap() error {
	return r.Err
}

func (r errorWithCode) StatusCode() int {
	return r.Code
}

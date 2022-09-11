package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"
)

type Client struct {
	endpoint string
	client   http.Client
}

// NewClient returns new API client.
func NewClient(endpoint string) *Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	c := Client{
		endpoint: endpoint,
		client: http.Client{
			Timeout: 5 * time.Second,
			Jar:     jar,
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

func (c *Client) CreateCompiler(ctx context.Context, form CreateCompilerForm) (Compiler, error) {
	if closer, ok := form.ImageFile.(io.Closer); ok {
		defer func() { _ = closer.Close() }()
	}
	buf := bytes.Buffer{}
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("name", form.Name); err != nil {
		return Compiler{}, err
	}
	if config, err := form.Config.MarshalJSON(); err != nil {
		return Compiler{}, err
	} else if err := w.WriteField("config", string(config)); err != nil {
		return Compiler{}, err
	}
	fileName := "unnamed.tar.gz"
	if f, ok := form.ImageFile.(*os.File); ok {
		fileName = f.Name()
	}
	if fw, err := w.CreateFormFile("file", fileName); err != nil {
		return Compiler{}, err
	} else if _, err := io.Copy(fw, form.ImageFile); err != nil {
		return Compiler{}, err
	}
	if err := w.Close(); err != nil {
		return Compiler{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.getURL("/v0/compilers"), &buf,
	)
	if err != nil {
		return Compiler{}, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	var respData Compiler
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *Client) ObserveRoles(ctx context.Context) (Roles, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.getURL("/v0/roles"), nil,
	)
	if err != nil {
		return Roles{}, err
	}
	var respData Roles
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *Client) CreateRole(
	ctx context.Context, name string,
) (Role, error) {
	data, err := json.Marshal(createRoleForm{
		Name: name,
	})
	if err != nil {
		return Role{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.getURL("/v0/roles"), bytes.NewReader(data),
	)
	if err != nil {
		return Role{}, err
	}
	var respData Role
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *Client) DeleteRole(
	ctx context.Context, name any,
) (Role, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete, c.getURL("/v0/roles/%v", name), nil,
	)
	if err != nil {
		return Role{}, err
	}
	var respData Role
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *Client) ObserveUserRoles(
	ctx context.Context, login string,
) (Roles, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet,
		c.getURL("/v0/users/%s/roles", login), nil,
	)
	if err != nil {
		return Roles{}, err
	}
	var respData Roles
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *Client) CreateUserRole(
	ctx context.Context, login string, role string,
) (Roles, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		c.getURL("/v0/users/%s/roles/%s", login, role), nil,
	)
	if err != nil {
		return Roles{}, err
	}
	var respData Roles
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *Client) getURL(path string, args ...any) string {
	return c.endpoint + fmt.Sprintf(path, args...)
}

func (c *Client) doRequest(req *http.Request, code int, respData any) (*http.Response, error) {
	if len(req.Header.Get("Content-Type")) == 0 {
		req.Header.Add("Content-Type", "application/json")
	}
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

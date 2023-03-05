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
	"net/url"
	"time"
)

type Client struct {
	endpoint string
	client   http.Client
	Headers  map[string]string
}

type ClientOption func(*Client)

func WithSessionCookie(value string) ClientOption {
	return func(c *Client) {
		u, err := url.Parse(c.endpoint)
		if err != nil {
			panic(err)
		}
		c.client.Jar.SetCookies(u, []*http.Cookie{{
			Name:  sessionCookie,
			Value: value,
		}})
	}
}

func WithTransport(transport *http.Transport) ClientOption {
	return func(c *Client) {
		c.client.Transport = transport
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.client.Timeout = timeout
	}
}

// NewClient returns new API client.
func NewClient(endpoint string, options ...ClientOption) *Client {
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
	for _, option := range options {
		option(&c)
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

func (c *Client) Locale(ctx context.Context) (Locale, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.getURL("/v0/locale"), nil,
	)
	if err != nil {
		return Locale{}, err
	}
	var respData Locale
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
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

func (c *Client) Logout(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.getURL("/v0/logout"), nil,
	)
	if err != nil {
		return err
	}
	_, err = c.doRequest(req, http.StatusOK, nil)
	return err
}

func (c *Client) Register(
	ctx context.Context, form RegisterUserForm,
) (User, error) {
	data, err := json.Marshal(form)
	if err != nil {
		return User{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.getURL("/v0/register"),
		bytes.NewReader(data),
	)
	if err != nil {
		return User{}, err
	}
	var respData User
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *Client) ObserveCompilers(ctx context.Context) (Compilers, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.getURL("/v0/compilers"), nil,
	)
	if err != nil {
		return Compilers{}, err
	}
	var respData Compilers
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *Client) CreateCompiler(ctx context.Context, form CreateCompilerForm) (Compiler, error) {
	defer func() { _ = form.Close() }()
	buf := bytes.Buffer{}
	w := multipart.NewWriter(&buf)
	if form.Name != nil {
		if err := w.WriteField("name", *form.Name); err != nil {
			return Compiler{}, err
		}
	}
	if form.Config.JSON != nil {
		if config, err := form.Config.MarshalJSON(); err != nil {
			return Compiler{}, err
		} else if err := w.WriteField("config", string(config)); err != nil {
			return Compiler{}, err
		}
	}
	if form.ImageFile != nil {
		if fw, err := w.CreateFormFile("file", form.ImageFile.Name); err != nil {
			return Compiler{}, err
		} else if _, err := io.Copy(fw, form.ImageFile.Reader); err != nil {
			return Compiler{}, err
		}
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

func (c *Client) UpdateCompiler(ctx context.Context, id int64, form UpdateCompilerForm) (Compiler, error) {
	defer func() { _ = form.Close() }()
	buf := bytes.Buffer{}
	w := multipart.NewWriter(&buf)
	if form.Name != nil {
		if err := w.WriteField("name", *form.Name); err != nil {
			return Compiler{}, err
		}
	}
	if form.Config.JSON != nil {
		if config, err := form.Config.MarshalJSON(); err != nil {
			return Compiler{}, err
		} else if err := w.WriteField("config", string(config)); err != nil {
			return Compiler{}, err
		}
	}
	if form.ImageFile != nil {
		if fw, err := w.CreateFormFile("file", form.ImageFile.Name); err != nil {
			return Compiler{}, err
		} else if _, err := io.Copy(fw, form.ImageFile.Reader); err != nil {
			return Compiler{}, err
		}
	}
	if err := w.Close(); err != nil {
		return Compiler{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPatch, c.getURL("/v0/compilers/%d", id), &buf,
	)
	if err != nil {
		return Compiler{}, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	var respData Compiler
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *Client) DeleteCompiler(ctx context.Context, id int64) (Compiler, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete, c.getURL("/v0/compilers/%d", id), nil,
	)
	if err != nil {
		return Compiler{}, err
	}
	var respData Compiler
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *Client) CreateProblem(ctx context.Context, form CreateProblemForm) (Problem, error) {
	defer func() { _ = form.Close() }()
	buf := bytes.Buffer{}
	w := multipart.NewWriter(&buf)
	if form.Title != nil {
		if err := w.WriteField("title", *form.Title); err != nil {
			return Problem{}, err
		}
	}
	if form.PackageFile != nil {
		if fw, err := w.CreateFormFile("file", form.PackageFile.Name); err != nil {
			return Problem{}, err
		} else if _, err := io.Copy(fw, form.PackageFile.Reader); err != nil {
			return Problem{}, err
		}
	}
	if err := w.Close(); err != nil {
		return Problem{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.getURL("/v0/problems"), &buf,
	)
	if err != nil {
		return Problem{}, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	var respData Problem
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *Client) UpdateProblem(ctx context.Context, id int64, form UpdateProblemForm) (Problem, error) {
	defer func() { _ = form.Close() }()
	buf := bytes.Buffer{}
	w := multipart.NewWriter(&buf)
	if form.Title != nil {
		if err := w.WriteField("title", *form.Title); err != nil {
			return Problem{}, err
		}
	}
	if form.PackageFile != nil {
		if fw, err := w.CreateFormFile("file", form.PackageFile.Name); err != nil {
			return Problem{}, err
		} else if _, err := io.Copy(fw, form.PackageFile.Reader); err != nil {
			return Problem{}, err
		}
	}
	if err := w.Close(); err != nil {
		return Problem{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPatch, c.getURL("/v0/problems/%d", id), &buf,
	)
	if err != nil {
		return Problem{}, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	var respData Problem
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *Client) DeleteProblem(ctx context.Context, id int64) (Problem, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete, c.getURL("/v0/problems/%d", id), nil,
	)
	if err != nil {
		return Problem{}, err
	}
	var respData Problem
	_, err = c.doRequest(req, http.StatusOK, &respData)
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

func (c *Client) SubmitContestSolution(
	ctx context.Context, contest int64, problem string, form SubmitSolutionForm,
) (ContestSolution, error) {
	defer func() { _ = form.ContentFile.Close() }()
	buf := bytes.Buffer{}
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("config", fmt.Sprint(form.CompilerID)); err != nil {
		return ContestSolution{}, err
	}
	if fw, err := w.CreateFormFile("file", form.ContentFile.Name); err != nil {
		return ContestSolution{}, err
	} else if _, err := io.Copy(fw, form.ContentFile.Reader); err != nil {
		return ContestSolution{}, err
	}
	if err := w.Close(); err != nil {
		return ContestSolution{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		c.getURL("/v0/contests/%d/problems/%s/submit", contest, problem),
		&buf,
	)
	if err != nil {
		return ContestSolution{}, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	var respData ContestSolution
	_, err = c.doRequest(req, http.StatusCreated, &respData)
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
) (Role, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		c.getURL("/v0/users/%s/roles/%s", login, role), nil,
	)
	if err != nil {
		return Role{}, err
	}
	var respData Role
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *Client) DeleteUserRole(
	ctx context.Context, login string, role string,
) (Role, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete,
		c.getURL("/v0/users/%s/roles/%s", login, role), nil,
	)
	if err != nil {
		return Role{}, err
	}
	var respData Role
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *Client) ObserveContest(
	ctx context.Context, id int64,
) (Contest, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.getURL("/v0/contests/%d", id), nil,
	)
	if err != nil {
		return Contest{}, err
	}
	var respData Contest
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *Client) ObserveContestSolutions(
	ctx context.Context, id int64,
) (ContestSolutions, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.getURL("/v0/contests/%d/solutions", id), nil,
	)
	if err != nil {
		return ContestSolutions{}, err
	}
	var respData ContestSolutions
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *Client) ObserveContestSolution(
	ctx context.Context, id int64, solutionID int64,
) (ContestSolution, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.getURL("/v0/contests/%d/solutions/%d", id, solutionID), nil,
	)
	if err != nil {
		return ContestSolution{}, err
	}
	var respData ContestSolution
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *Client) CreateContestParticipant(
	ctx context.Context,
	contest int64,
	form CreateContestParticipantForm,
) (ContestParticipant, error) {
	data, err := json.Marshal(form)
	if err != nil {
		return ContestParticipant{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		c.getURL("/v0/contests/%d/participants", contest),
		bytes.NewReader(data),
	)
	if err != nil {
		return ContestParticipant{}, err
	}
	var respData ContestParticipant
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *Client) ObserveSettings(ctx context.Context) (Settings, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.getURL("/v0/settings"), nil,
	)
	if err != nil {
		return Settings{}, err
	}
	var respData Settings
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *Client) ObserveScopeUsers(ctx context.Context, scope int64) (ScopeUsers, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.getURL("/v0/scopes/%d/users", scope), nil,
	)
	if err != nil {
		return ScopeUsers{}, err
	}
	var respData ScopeUsers
	_, err = c.doRequest(req, http.StatusOK, &respData)
	return respData, err
}

func (c *Client) CreateSetting(ctx context.Context, form CreateSettingForm) (Setting, error) {
	data, err := json.Marshal(form)
	if err != nil {
		return Setting{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		c.getURL("/v0/settings"), bytes.NewReader(data),
	)
	if err != nil {
		return Setting{}, err
	}
	var respData Setting
	_, err = c.doRequest(req, http.StatusCreated, &respData)
	return respData, err
}

func (c *Client) UpdateSetting(ctx context.Context, id int64, form UpdateSettingForm) (Setting, error) {
	data, err := json.Marshal(form)
	if err != nil {
		return Setting{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPatch,
		c.getURL("/v0/settings/%d", id), bytes.NewReader(data),
	)
	if err != nil {
		return Setting{}, err
	}
	var respData Setting
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
	for key, value := range c.Headers {
		req.Header.Add(key, value)
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

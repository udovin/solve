package api

import (
	"net/http"
	"time"

	"github.com/udovin/solve/internal/api"
	"github.com/udovin/solve/internal/models"
)

type (
	ClientOption = api.ClientOption

	CreateCompilerForm           = api.CreateCompilerForm
	CreateContestParticipantForm = api.CreateContestParticipantForm
	CreateProblemForm            = api.CreateProblemForm
	CreateSettingForm            = api.CreateSettingForm
	RegisterUserForm             = api.RegisterUserForm
	SubmitSolutionForm           = api.SubmitSolutionForm
	UpdateCompilerForm           = api.UpdateCompilerForm
	UpdateProblemForm            = api.UpdateProblemForm
	UpdateSettingForm            = api.UpdateSettingForm

	Compiler           = api.Compiler
	Compilers          = api.Compilers
	Contest            = api.Contest
	ContestParticipant = api.ContestParticipant
	ContestSolution    = api.ContestSolution
	ContestSolutions   = api.ContestSolutions
	ContestStandings   = api.ContestStandings
	FileReader         = api.FileReader
	Problem            = api.Problem
	Role               = api.Role
	Roles              = api.Roles
	ScopeUser          = api.ScopeUser
	ScopeUsers         = api.ScopeUsers
	Setting            = api.Setting
	Settings           = api.Settings
	Solution           = api.Solution
	User               = api.User

	CompilerConfig = models.CompilerConfig
)

const (
	RegularParticipant   = models.RegularParticipant
	UpsolvingParticipant = models.UpsolvingParticipant
	ManagerParticipant   = models.ManagerParticipant
	ObserverParticipant  = models.ObserverParticipant
)

type Client struct {
	*api.Client
}

func WithSessionCookie(value string) ClientOption {
	return api.WithSessionCookie(value)
}

func WithTransport(transport *http.Transport) ClientOption {
	return api.WithTransport(transport)
}

func WithTimeout(timeout time.Duration) ClientOption {
	return api.WithTimeout(timeout)
}

// NewClient returns new solve API client.
func NewClient(endpoint string, options ...ClientOption) *Client {
	return &Client{
		Client: api.NewClient(endpoint, options...),
	}
}

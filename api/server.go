package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/labstack/echo"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

type Server struct {
	app *core.App
}

const (
	userKey    = "FullUser"
	sessionKey = "Session"
	visitKey   = "Visit"
	rolesKey   = "Roles"
)

func Register(app *core.App, api *echo.Group) {
	v := Server{app: app}
	api.Use(v.logVisit)
	// Service handlers
	api.GET("/ping", v.Ping)
	// Users management
	// api.POST("/users", v.CreateUser)
	api.GET(
		"/users/:UserID", v.GetUser,
		v.requireAuth(v.sessionAuth),
	)
	api.GET(
		"/users/:UserID/sessions", v.GetUserSessions,
		v.requireAuth(v.sessionAuth),
	)
	// api.PATCH(
	// 	"/users/:UserID", v.UpdateUser,
	// 	v.requireAuth(v.sessionAuth),
	// )
	// api.DELETE(
	// 	"/users/:UserID", v.DeleteUser,
	// 	v.requireAuth(v.sessionAuth),
	// )
	// Sessions management
	api.GET(
		"/sessions", v.GetSessions,
		v.requireAuth(v.sessionAuth),
	)
	api.GET(
		"/sessions/current", v.GetCurrentSession,
		v.requireAuth(v.sessionAuth),
	)
	api.POST(
		"/sessions", v.CreateSession,
		v.requireAuth(v.passwordAuth),
	)
	api.PATCH(
		"/sessions/:SessionID", v.UpdateSession,
		v.requireAuth(v.sessionAuth),
	)
	api.DELETE(
		"/sessions/:SessionID", v.DeleteSession,
		v.requireAuth(v.sessionAuth),
	)
	// Problems management
	api.POST(
		"/problems", v.CreateProblem,
		v.requireAuth(v.sessionAuth),
	)
	api.GET(
		"/problems/:ProblemID", v.GetProblem,
		v.requireAuth(v.sessionAuth),
	)
	api.PATCH(
		"/problems/:ProblemID", v.UpdateProblem,
		v.requireAuth(v.sessionAuth),
	)
	// Contests management
	api.GET(
		"/contests", v.GetContests,
		v.requireAuth(v.sessionAuth),
	)
	api.POST(
		"/contests", v.CreateContest,
		v.requireAuth(v.sessionAuth),
	)
	api.GET(
		"/contests/:ContestID", v.GetContest,
		v.requireAuth(v.sessionAuth),
	)
	api.PATCH(
		"/contests/:ContestID", v.UpdateContest,
		v.requireAuth(v.sessionAuth),
	)
	api.GET(
		"/contests/:ContestID/solutions", v.GetContestSolutions,
		v.requireAuth(v.sessionAuth),
	)
	api.POST(
		"/contests/:ContestID/problems", v.CreateContestProblem,
		v.requireAuth(v.sessionAuth),
	)
	api.GET(
		"/contests/:ContestID/problems/:ProblemCode",
		v.GetContestProblem, v.requireAuth(v.sessionAuth),
	)
	api.POST(
		"/contests/:ContestID/problems/:ProblemCode",
		v.CreateContestSolution, v.requireAuth(v.sessionAuth),
	)
	// Compilers management
	api.GET(
		"/compilers", v.GetCompilers,
		v.requireAuth(v.sessionAuth),
	)
	api.POST(
		"/compilers", v.CreateCompiler,
		v.requireAuth(v.sessionAuth),
	)
	api.GET(
		"/compilers/:CompilerID", v.GetCompiler,
		v.requireAuth(v.sessionAuth),
	)
	// Solutions management
	api.GET(
		"/solutions", v.GetSolutions,
		v.requireAuth(v.sessionAuth),
	)
	api.GET(
		"/solutions/:SolutionID", v.GetSolution,
		v.requireAuth(v.sessionAuth),
	)
	api.POST(
		"/solutions/:SolutionID", v.RejudgeSolution,
		v.requireAuth(v.sessionAuth),
	)
	api.POST(
		"/solutions/:SolutionID/report", v.createSolutionReport,
		v.requireAuth(v.sessionAuth),
	)
	// Participants management
	api.POST(
		"/participants", v.CreateParticipant,
		v.requireAuth(v.sessionAuth),
	)
}

func (s *Server) Ping(c echo.Context) error {
	return c.JSON(http.StatusOK, "pong")
}

// NewServer returns a new instance of server.
func NewServer(app *core.App) *Server {
	return &Server{app: app}
}

// Register registers handlers in specified group.
func (s *Server) Register(g *echo.Group) {
	g.Use(s.logVisit)
	s.registerUserHandlers(g)
}

// logVisit saves visit to visit store.
func (s *Server) logVisit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set(visitKey, s.app.Visits.MakeFromContext(c))
		defer func() {
			visit := c.Get(visitKey).(models.Visit)
			if user, ok := c.Get(userKey).(models.User); ok {
				visit.UserID = models.NInt64(user.ID)
			}
			if session, ok := c.Get(sessionKey).(models.Session); ok {
				visit.SessionID = models.NInt64(session.ID)
			}
			visit.Status = c.Response().Status
			if _, err := s.app.Visits.Create(visit); err != nil {
				c.Logger().Error(err)
			}
		}()
		return next(c)
	}
}

type authMethod func(echo.Context) error

var errNoAuth = errors.New("bad auth")

func (s *Server) requireAuth(methods ...authMethod) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			for _, method := range methods {
				if err := method(c); err != nil {
					if err == errNoAuth {
						continue
					}
					c.Logger().Error(err)
					return err
				}
				if err := s.extractRoles(c); err != nil {
					c.Logger().Error(err)
					return err
				}
				return next(c)
			}
			return c.NoContent(http.StatusForbidden)
		}
	}
}

func (s *Server) extractRoles(c echo.Context) error {
	if user, ok := c.Get(userKey).(models.User); ok {
		roles, err := s.app.GetUserRoles(user.ID)
		if err != nil {
			return err
		}
		c.Set(rolesKey, roles)
	} else {
		roles, err := s.app.GetGuestRoles()
		if err != nil {
			return err
		}
		c.Set(rolesKey, roles)
	}
	return nil
}

// guestAuth authorizes guest.
func (s *Server) guestAuth(c echo.Context) error {
	return nil
}

// sessionAuth tries to auth using session cookie.
func (s *Server) sessionAuth(c echo.Context) error {
	cookie, err := c.Cookie(sessionKey)
	if err != nil {
		return errNoAuth
	}
	session, err := s.app.Sessions.GetByCookie(cookie.Value)
	if err != nil {
		return errNoAuth
	}
	user, err := s.app.Users.Get(session.UserID)
	if err != nil {
		return errNoAuth
	}
	c.Set(userKey, user)
	c.Set(sessionKey, session)
	return nil
}

// passwordAuth tries to auth using login and password.
func (s *Server) passwordAuth(c echo.Context) error {
	var authData struct {
		Login    string `json:""`
		Password string `json:""`
	}
	if err := c.Bind(&authData); err != nil {
		return errNoAuth
	}
	user, err := s.app.Users.GetByLogin(authData.Login)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNoAuth
		}
		return err
	}
	if !s.app.Users.CheckPassword(user, authData.Password) {
		return errNoAuth
	}
	c.Set(userKey, user)
	return nil
}

// requireRole check that user has required roles.
func (s *Server) requireRole(code string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			roles, ok := c.Get(rolesKey).(core.Roles)
			if !ok {
				return c.NoContent(http.StatusForbidden)
			}
			ok, err := s.app.HasRole(roles, code)
			if err != nil {
				return c.NoContent(http.StatusInternalServerError)
			}
			if !ok {
				return c.NoContent(http.StatusForbidden)
			}
			return next(c)
		}
	}
}

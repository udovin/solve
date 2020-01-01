package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

type View struct {
	app *core.App
}

type authMethod func(ctx echo.Context) error

var errBadAuth = errors.New("bad auth")

const (
	userKey    = "User"
	sessionKey = "Session"
)

func (v *View) authMiddleware(methods ...authMethod) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			for _, method := range methods {
				if err := method(c); err != nil {
					if err == errBadAuth {
						continue
					}
					c.Logger().Error(err)
					return err
				}
				return next(c)
			}
			return c.NoContent(http.StatusForbidden)
		}
	}
}

func (v *View) sessionAuth(c echo.Context) error {
	cookie, err := c.Cookie(sessionKey)
	if err != nil {
		return errBadAuth
	}
	session, err := v.app.Sessions.GetByCookie(cookie.Value)
	if err != nil {
		return errBadAuth
	}
	user, err := v.app.Users.Get(session.UserID)
	if err != nil {
		return errBadAuth
	}
	c.Set(userKey, user)
	c.Set(sessionKey, session)
	return nil
}

func (v *View) passwordAuth(c echo.Context) error {
	var authData struct {
		Login    string `json:""`
		Password string `json:""`
	}
	if err := c.Bind(&authData); err != nil {
		return errBadAuth
	}
	user, err := v.app.Users.GetByLogin(authData.Login)
	if err != nil ||
		!user.CheckPassword(authData.Password, v.app.PasswordSalt) {
		return errBadAuth
	}
	c.Set(userKey, user)
	return nil
}

const visitContext = "Visit"

func (v *View) logVisit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		visit := models.Visit{
			Time: time.Now().Unix(),
		}
		c.Set(visitContext, visit)
		defer func() {
			visit := c.Get(visitContext).(models.Visit)
			_, _ = v.app.Visits.Create(visit)
		}()
		return next(c)
	}
}

func Register(app *core.App, api *echo.Group) {
	v := View{app: app}
	api.Use(v.logVisit)
	// Service handlers
	api.GET("/ping", v.Ping)
	// Users management
	api.POST("/users", v.CreateUser)
	api.GET(
		"/users/:UserID", v.GetUser,
		v.authMiddleware(v.sessionAuth),
	)
	api.GET(
		"/users/:UserID/sessions", v.GetUserSessions,
		v.authMiddleware(v.sessionAuth),
	)
	api.PATCH(
		"/users/:UserID", v.UpdateUser,
		v.authMiddleware(v.sessionAuth),
	)
	api.DELETE(
		"/users/:UserID", v.DeleteUser,
		v.authMiddleware(v.sessionAuth),
	)
	// Sessions management
	api.GET(
		"/sessions", v.GetSessions,
		v.authMiddleware(v.sessionAuth),
	)
	api.GET(
		"/sessions/current", v.GetCurrentSession,
		v.authMiddleware(v.sessionAuth),
	)
	api.POST(
		"/sessions", v.CreateSession,
		v.authMiddleware(v.passwordAuth),
	)
	api.PATCH(
		"/sessions/:SessionID", v.UpdateSession,
		v.authMiddleware(v.sessionAuth),
	)
	api.DELETE(
		"/sessions/:SessionID", v.DeleteSession,
		v.authMiddleware(v.sessionAuth),
	)
	// Problems management
	api.POST(
		"/problems", v.CreateProblem,
		v.authMiddleware(v.sessionAuth),
	)
	api.GET(
		"/problems/:ProblemID", v.GetProblem,
		v.authMiddleware(v.sessionAuth),
	)
	api.PATCH(
		"/problems/:ProblemID", v.UpdateProblem,
		v.authMiddleware(v.sessionAuth),
	)
	// Contests management
	api.GET(
		"/contests", v.GetContests,
		v.authMiddleware(v.sessionAuth),
	)
	api.POST(
		"/contests", v.CreateContest,
		v.authMiddleware(v.sessionAuth),
	)
	api.GET(
		"/contests/:ContestID", v.GetContest,
		v.authMiddleware(v.sessionAuth),
	)
	api.PATCH(
		"/contests/:ContestID", v.UpdateContest,
		v.authMiddleware(v.sessionAuth),
	)
	api.GET(
		"/contests/:ContestID/solutions", v.GetContestSolutions,
		v.authMiddleware(v.sessionAuth),
	)
	api.POST(
		"/contests/:ContestID/problems", v.CreateContestProblem,
		v.authMiddleware(v.sessionAuth),
	)
	api.GET(
		"/contests/:ContestID/problems/:ProblemCode",
		v.GetContestProblem, v.authMiddleware(v.sessionAuth),
	)
	api.POST(
		"/contests/:ContestID/problems/:ProblemCode",
		v.CreateContestSolution, v.authMiddleware(v.sessionAuth),
	)
	// Compilers management
	api.GET(
		"/compilers", v.GetCompilers,
		v.authMiddleware(v.sessionAuth),
	)
	api.POST(
		"/compilers", v.CreateCompiler,
		v.authMiddleware(v.sessionAuth),
	)
	api.GET(
		"/compilers/:CompilerID", v.GetCompiler,
		v.authMiddleware(v.sessionAuth),
	)
	// Solutions management
	api.GET(
		"/solutions", v.GetSolutions,
		v.authMiddleware(v.sessionAuth),
	)
	api.GET(
		"/solutions/:SolutionID", v.GetSolution,
		v.authMiddleware(v.sessionAuth),
	)
	api.POST(
		"/solutions/:SolutionID", v.RejudgeSolution,
		v.authMiddleware(v.sessionAuth),
	)
	api.POST(
		"/solutions/:SolutionID/report", v.createSolutionReport,
		v.authMiddleware(v.sessionAuth),
	)
	// Participants management
	api.POST(
		"/participants", v.CreateParticipant,
		v.authMiddleware(v.sessionAuth),
	)
}

func (v *View) Ping(c echo.Context) error {
	return c.JSON(http.StatusOK, "pong")
}

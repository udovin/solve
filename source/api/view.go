package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo"

	"../core"
)

type View struct {
	app    *core.App
	server *echo.Echo
}

type authMethod func(ctx echo.Context) error

var badAuthError = errors.New("bad auth")

const (
	userKey    = "User"
	sessionKey = "Session"
)

func (v *View) authMiddleware(methods ...authMethod) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			for _, method := range methods {
				if err := method(ctx); err != nil {
					if err == badAuthError {
						continue
					}
					return err
				}
				return next(ctx)
			}
			return badAuthError
		}
	}
}

func (v *View) sessionAuth(ctx echo.Context) error {
	return nil
}

func (v *View) passwordAuth(ctx echo.Context) error {
	return nil
}

func Register(app *core.App, server *echo.Echo) {
	v := View{app: app, server: server}
	// Service handlers
	server.GET("/api/v0/ping", v.Ping)
	// Users management
	server.POST("/api/v0/users", v.CreateUser)
	server.GET(
		"/api/v0/users/:UserID", v.GetUser,
		v.authMiddleware(v.sessionAuth),
	)
	server.PATCH(
		"/api/v0/users/:UserID", v.UpdateUser,
		v.authMiddleware(v.sessionAuth),
	)
	server.DELETE(
		"/api/v0/users/:UserID", v.DeleteUser,
		v.authMiddleware(v.sessionAuth),
	)
	// Sessions management
	server.POST(
		"/api/v0/sessions", v.CreateSession,
		v.authMiddleware(v.passwordAuth),
	)
	server.PATCH(
		"/api/v0/sessions/:SessionID", v.UpdateSession,
		v.authMiddleware(v.sessionAuth),
	)
	// Problems management
	server.POST(
		"/api/v0/problems", v.CreateProblem,
		v.authMiddleware(v.sessionAuth),
	)
	server.GET(
		"/api/v0/problems/:ProblemID", v.GetProblem,
		v.authMiddleware(v.sessionAuth),
	)
	server.PATCH(
		"/api/v0/problems/:ProblemID", v.UpdateProblem,
		v.authMiddleware(v.sessionAuth),
	)
	// Contests management
	server.POST(
		"/api/v0/contests", v.CreateContest,
		v.authMiddleware(v.sessionAuth),
	)
	server.GET(
		"/api/v0/contests/:ContestID", v.GetContest,
		v.authMiddleware(v.sessionAuth),
	)
	server.PATCH(
		"/api/v0/contests/:ContestID", v.UpdateContest,
		v.authMiddleware(v.sessionAuth),
	)
}

func (v *View) Ping(c echo.Context) error {
	return c.JSON(http.StatusOK, "pong")
}

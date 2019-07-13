package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo"

	"github.com/udovin/solve/core"
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
		return func(c echo.Context) error {
			for _, method := range methods {
				if err := method(c); err != nil {
					if err == badAuthError {
						continue
					}
					return err
				}
				return next(c)
			}
			return badAuthError
		}
	}
}

func (v *View) sessionAuth(c echo.Context) error {
	cookie, err := c.Cookie(sessionKey)
	if err != nil {
		return badAuthError
	}
	session, ok := v.app.SessionStore.GetByCookie(cookie.Value)
	if !ok {
		return badAuthError
	}
	user, ok := v.app.UserStore.Get(session.UserID)
	if !ok {
		return badAuthError
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
		return badAuthError
	}
	user, ok := v.app.UserStore.GetByLogin(authData.Login)
	if !ok || !user.CheckPassword(authData.Password, v.app.PasswordSalt) {
		return badAuthError
	}
	c.Set(userKey, user)
	return nil
}

func Register(app *core.App, server *echo.Echo) {
	v := View{app: app, server: server}
	// Create group for api handlers
	api := server.Group("/api/v0")
	// Service handlers
	api.GET("/ping", v.Ping)
	// Users management
	api.POST("/users", v.CreateUser)
	api.GET(
		"/users/:UserID", v.GetUser,
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
		"/sessions", v.GetSessionList,
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
		"/contests", v.GetContestList,
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
}

func (v *View) Ping(c echo.Context) error {
	return c.JSON(http.StatusOK, "pong")
}

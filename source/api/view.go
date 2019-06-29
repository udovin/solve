package api

import (
	"net/http"

	"github.com/labstack/echo"

	"../core"
)

type View struct {
	app    *core.App
	server *echo.Echo
}

func Register(app *core.App, server *echo.Echo) {
	v := View{app: app, server: server}
	// Service handlers
	server.GET("/api/v0/ping", v.Ping)
	// Users management
	server.POST("/api/v0/users", v.CreateUser)
	server.GET("/api/v0/users/:UserID", v.GetUser)
	server.PATCH("/api/v0/users/:UserID", v.UpdateUser)
	server.DELETE("/api/v0/users/:UserID", v.DeleteUser)
	// Sessions management
	server.POST("/api/v0/sessions", v.CreateSession)
	server.PATCH("/api/v0/sessions/:SessionID", v.UpdateSession)
	// Problems management
	server.POST("/api/v0/problems", v.CreateProblem)
	server.GET("/api/v0/problems/:ProblemID", v.GetProblem)
	server.PATCH("/api/v0/problems/:ProblemID", v.UpdateProblem)
	// Contests management
	server.POST("/api/v0/contests", v.CreateContest)
	server.GET("/api/v0/contests/:ContestID", v.GetContest)
	server.PATCH("/api/v0/contests/:ContestID", v.UpdateContest)
}

func (v *View) Ping(c echo.Context) error {
	return c.JSON(http.StatusOK, "pong")
}

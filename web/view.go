package web

import (
	"github.com/labstack/echo"

	"../core"
)

func indexPage(c echo.Context) error {
	return c.File("index.html")
}

func Register(app *core.App, server *echo.Echo) {
	server.Static("/static/*", "static")
	server.File("/favicon.ico", "favicon.ico")
	server.Any("/*", indexPage)
}

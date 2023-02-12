package api

import (
	"github.com/labstack/echo/v4"

	"github.com/udovin/solve/models"
)

func (v *View) registerContestMessageHandlers(g *echo.Group) {
	g.GET(
		"/v0/contests/:contest/messages", v.observeContestStandings,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractContest,
		v.requirePermission(models.ObserveContestMessagesRole),
	)
}

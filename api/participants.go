package api

import (
	"net/http"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

func (v *View) CreateParticipant(c echo.Context) error {
	var participant models.Participant
	if err := c.Bind(&participant); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	user, ok := c.Get(authUserKey).(models.User)
	if !ok {
		return c.NoContent(http.StatusForbidden)
	}
	if !user.IsSuper {
		return c.NoContent(http.StatusForbidden)
	}
	if err := v.core.Participants.Create(&participant); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, participant)
}

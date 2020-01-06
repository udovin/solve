package api

import (
	"database/sql"
	"net/http"
	"sort"
	"strconv"

	"github.com/labstack/echo"

	"github.com/udovin/solve/models"
)

type session struct {
	models.Session
	User models.User `json:""`
}

func (v *View) listSessions(c echo.Context) error {
	user := c.Get(authUserKey).(models.User)
	sessions, err := v.core.Sessions.FindByUser(user.ID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	sort.Sort(sessionModelSorter(sessions))
	return c.JSON(http.StatusOK, sessions)
}

func (v *View) deleteSession(c echo.Context) error {
	session := c.Get(sessionKey).(models.Session)
	if err := v.core.WithTx(func(tx *sql.Tx) error {
		return v.core.Sessions.DeleteTx(tx, session.ID)
	}); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusOK)
}

const sessionKey = "Session"

// extractSession extracts session from ID.
func (v *View) extractSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("SessionID"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return c.NoContent(http.StatusBadRequest)
		}
		session, err := v.core.Sessions.Get(id)
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		c.Set(sessionKey, session)
		return next(c)
	}
}

type sessionModelSorter []models.Session

func (c sessionModelSorter) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c sessionModelSorter) Len() int {
	return len(c)
}

func (c sessionModelSorter) Less(i, j int) bool {
	return c[i].ID > c[j].ID
}

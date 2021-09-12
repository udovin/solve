package api

import (
	"database/sql"
	"fmt"
	"github.com/labstack/echo"
	"github.com/udovin/solve/models"
	"net/http"
	"sort"
	"strconv"
)

func (v *View) registerContestHandlers(g *echo.Group) {
	g.GET(
		"/contests", v.observeContests,
		v.sessionAuth,
	)
	g.GET(
		"/contests/:contest", v.observeContest,
		v.sessionAuth, v.requireAuth, v.extractContest,
		v.requireAuthRole(models.ObserveContestRole),
	)
}

type Contest struct {
	ID int64 `json:"id"`
}

type contestIDDescSorter []Contest

func (v contestIDDescSorter) Len() int {
	return len(v)
}

func (v contestIDDescSorter) Less(i, j int) bool {
	return v[i].ID > v[j].ID
}

func (v contestIDDescSorter) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v *View) observeContests(c echo.Context) error {
	var resp []Contest
	contests, err := v.core.Contests.All()
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	for _, contest := range contests {
		resp = append(resp, Contest{
			ID: contest.ID,
		})
	}
	sort.Sort(contestIDDescSorter(resp))
	return c.JSON(http.StatusOK, resp)
}

func (v *View) observeContest(c echo.Context) error {
	contest, ok := c.Get(contestKey).(models.Contest)
	if !ok {
		c.Logger().Error("contest not extracted")
		return fmt.Errorf("contest not extracted")
	}
	resp := Contest{
		ID: contest.ID,
	}
	return c.JSON(http.StatusOK, resp)
}

func (v *View) extractContest(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id, err := strconv.ParseInt(c.Param("contest"), 10, 64)
		if err != nil {
			c.Logger().Warn(err)
			return err
		}
		contest, err := v.core.Contests.Get(id)
		if err != nil {
			if err == sql.ErrNoRows {
				resp := errorResp{Message: "contest not found"}
				return c.JSON(http.StatusNotFound, resp)
			}
			c.Logger().Error(err)
			return err
		}
		c.Set(contestKey, contest)
		return next(c)
	}
}

package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

func (v *View) registerContestStandingsHandlers(g *echo.Group) {
	g.GET(
		"/v0/contests/:contest/standings", v.observeContestStandings,
		v.extractAuth(v.sessionAuth, v.guestAuth), v.extractContest,
		v.requirePermission(models.ObserveContestStandingsRole),
	)
}

type ContestStandingsColumn struct {
	Code string `json:"code"`
}

type ContestStandingsCell struct {
	Column  int    `json:"column"`
	Verdict string `json:"verdict,omitempty"`
	Attempt int    `json:"attempt,omitempty"`
	Time    int64  `json:"time,omitempty"`
}

type ContestStandingsRow struct {
	Participant ContestParticipant     `json:"participant,omitempty"`
	Score       int                    `json:"score,omitempty"`
	Penalty     int64                  `json:"penalty,omitempty"`
	Cells       []ContestStandingsCell `json:"cells,omitempty"`
}

type ContestStandings struct {
	Columns []ContestStandingsColumn `json:"columns"`
	Rows    []ContestStandingsRow    `json:"rows"`
}

func (v *View) observeContestStandings(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	standings, err := v.standings.BuildStandings(getContext(c), contest)
	if err != nil {
		return err
	}
	resp := ContestStandings{}
	for _, column := range standings.Columns {
		resp.Columns = append(resp.Columns, ContestStandingsColumn{
			Code: column.Problem.Code,
		})
	}
	for _, row := range standings.Rows {
		rowResp := ContestStandingsRow{
			Participant: makeContestParticipant(row.Participant, v.core),
			Score:       row.Score,
			Penalty:     row.Penalty,
		}
		for _, cell := range row.Cells {
			cellResp := ContestStandingsCell{
				Column:  cell.Column,
				Attempt: cell.Attempt,
				Time:    cell.Time,
			}
			if cell.Verdict != 0 {
				if cell.Verdict == models.Accepted {
					cellResp.Verdict = models.Accepted.String()
				} else {
					cellResp.Verdict = models.Rejected.String()
				}
			}
			rowResp.Cells = append(rowResp.Cells, cellResp)
		}
		resp.Rows = append(resp.Rows, rowResp)
	}
	return c.JSON(http.StatusOK, resp)
}

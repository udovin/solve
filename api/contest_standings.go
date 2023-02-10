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
	Code   string `json:"code"`
	Points *int   `json:"points,omitempty"`
}

type ContestStandingsCell struct {
	Column  int    `json:"column"`
	Verdict string `json:"verdict"`
	Attempt int    `json:"attempt"`
	Time    *int64 `json:"time,omitempty"`
}

type ContestStandingsRow struct {
	Participant ContestParticipant     `json:"participant,omitempty"`
	Score       int                    `json:"score"`
	Penalty     *int64                 `json:"penalty,omitempty"`
	Cells       []ContestStandingsCell `json:"cells,omitempty"`
}

type ContestStandings struct {
	Columns []ContestStandingsColumn `json:"columns,omitempty"`
	Rows    []ContestStandingsRow    `json:"rows,omitempty"`
}

func (v *View) observeContestStandings(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	standings, err := v.standings.BuildStandings(getContext(c), contest, contestCtx.Now)
	if err != nil {
		return err
	}
	resp := ContestStandings{}
	for _, column := range standings.Columns {
		columnResp := ContestStandingsColumn{
			Code: column.Problem.Code,
		}
		config, err := column.Problem.GetConfig()
		if err == nil && config.Points != nil {
			columnResp.Points = config.Points
		}
		resp.Columns = append(resp.Columns, columnResp)
	}
	observeFullStandings := contestCtx.HasPermission(models.ObserveContestFullStandingsRole)
	for _, row := range standings.Rows {
		if !observeFullStandings {
			switch row.Participant.Kind {
			case models.RegularParticipant:
			case models.UpsolvingParticipant:
				if contestCtx.Stage != managers.ContestFinished {
					continue
				}
			default:
				continue
			}
		}
		rowResp := ContestStandingsRow{
			Participant: makeContestParticipant(c, row.Participant, v.core),
			Score:       row.Score,
		}
		if row.Participant.Kind == models.RegularParticipant {
			rowResp.Penalty = getPtr(row.Penalty)
		}
		for _, cell := range row.Cells {
			cellResp := ContestStandingsCell{
				Column:  cell.Column,
				Attempt: cell.Attempt,
			}
			if row.Participant.Kind == models.RegularParticipant {
				cellResp.Time = getPtr(cell.Time)
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

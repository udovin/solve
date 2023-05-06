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
	Code              string `json:"code"`
	Points            *int   `json:"points,omitempty"`
	TotalSolutions    int    `json:"total_solutions,omitempty"`
	AcceptedSolutions int    `json:"accepted_solutions,omitempty"`
}

type ContestStandingsCell struct {
	Column  int     `json:"column"`
	Verdict string  `json:"verdict"`
	Points  float64 `json:"points,omitempty"`
	Attempt int     `json:"attempt"`
	Time    *int64  `json:"time,omitempty"`
}

type ContestStandingsRow struct {
	Participant ContestParticipant     `json:"participant,omitempty"`
	Score       float64                `json:"score"`
	Penalty     *int64                 `json:"penalty,omitempty"`
	Place       int                    `json:"place,omitempty"`
	Cells       []ContestStandingsCell `json:"cells,omitempty"`
}

type ContestStandings struct {
	Kind    string                   `json:"kind"`
	Columns []ContestStandingsColumn `json:"columns,omitempty"`
	Rows    []ContestStandingsRow    `json:"rows,omitempty"`
	Frozen  bool                     `json:"frozen,omitempty"`
}

type ObserveContestStandingsForm struct {
	IgnoreFreeze bool `query:"ignore_freeze"`
	OnlyOfficial bool `query:"only_official"`
}

func (v *View) observeContestStandings(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	if contestCtx.ContestConfig.StandingsKind == models.DisabledStandings {
		return c.JSON(http.StatusOK, ContestStandings{
			Kind: contestCtx.ContestConfig.StandingsKind.String(),
		})
	}
	form := ObserveContestStandingsForm{}
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return errorResponse{
			Code:    http.StatusBadRequest,
			Message: localize(c, "Invalid form."),
		}
	}
	options := managers.BuildStandingsOptions{
		IgnoreFreeze: form.IgnoreFreeze,
		OnlyOfficial: form.OnlyOfficial,
	}
	standings, err := v.standings.BuildStandings(contestCtx, options)
	if err != nil {
		return err
	}
	resp := ContestStandings{
		Kind:   contestCtx.ContestConfig.StandingsKind.String(),
		Frozen: standings.Frozen,
	}
	for _, column := range standings.Columns {
		columnResp := ContestStandingsColumn{
			Code:              column.Problem.Code,
			TotalSolutions:    column.TotalSolutions,
			AcceptedSolutions: column.AcceptedSolutions,
		}
		config, err := column.Problem.GetConfig()
		if err == nil && config.Points != nil {
			columnResp.Points = config.Points
		}
		resp.Columns = append(resp.Columns, columnResp)
	}
	for _, row := range standings.Rows {
		rowResp := ContestStandingsRow{
			Participant: makeContestParticipant(c, row.Participant, v.core),
			Score:       row.Score,
			Penalty:     row.Penalty,
			Place:       row.Place,
		}
		for _, cell := range row.Cells {
			cellResp := ContestStandingsCell{
				Column:  cell.Column,
				Attempt: cell.Attempt,
				Points:  cell.Points,
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

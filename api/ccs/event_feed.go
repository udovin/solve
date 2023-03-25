package ccs

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

func (v *View) getEventFeed(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contestSolutions, err := v.core.ContestSolutions.FindByContest(contestCtx.Contest.ID)
	if err != nil {
		return err
	}
	closable := contestCtx.Stage == managers.ContestFinished
	events := []EventData{}
	for _, contestSolution := range contestSolutions {
		solution, err := v.core.Solutions.Get(c.Request().Context(), contestSolution.SolutionID)
		if err != nil {
			closable = false
			continue
		}
		events = append(events, Submission{
			ID:     ID(solution.ID),
			TeamID: ID(contestSolution.ParticipantID),
		})
		report, err := solution.GetReport()
		if err != nil {
			closable = false
			continue
		}
		if report == nil || report.Verdict == models.Failed {
			closable = false
			continue
		}
		events = append(events, Judgement{
			ID:           ID(solution.ID),
			SubmissionID: ID(solution.ID),
		})
	}
	c.Response().WriteHeader(http.StatusOK)
	for _, eventData := range events {
		event := Event{
			Type: eventData.Kind(),
			ID:   eventData.ObjectID(),
			Data: eventData,
		}
		bytes, err := json.Marshal(event)
		if err != nil {
			return err
		}
		bytes = append(bytes, '\n')
		if _, err := c.Response().Write(bytes); err != nil {
			return err
		}
	}
	c.Response().Flush()
	events = nil
	_ = closable
	return nil
}

type Event struct {
	Type string    `json:"type"`
	ID   ID        `json:"id"`
	Data EventData `json:"data"`
}

type EventData interface {
	Kind() string
	ObjectID() ID
}

type ID int64

func (v ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprint(v))
}

var _ json.Marshaler = ID(0)

type Submission struct {
	ID     ID `json:"id"`
	TeamID ID `json:"team_id"`
}

func (e Submission) Kind() string {
	return "submission"
}

func (e Submission) ObjectID() ID {
	return e.ID
}

type Judgement struct {
	ID           ID `json:"id"`
	SubmissionID ID `json:"submission_id"`
}

func (e Judgement) Kind() string {
	return "judgement"
}

func (e Judgement) ObjectID() ID {
	return e.ID
}

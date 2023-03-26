package ccs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

func (v *View) getEventFeed(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	if !contestCtx.HasPermission(models.ObserveContestStandingsRole) {
		return c.NoContent(http.StatusForbidden)
	}
	config := contestCtx.ContestConfig
	contestStart := time.Unix(int64(config.BeginTime), 0)
	contestFinish := contestStart.Add(time.Second * time.Duration(config.Duration))
	events := []EventData{}
	events = append(events, Contest{
		ID:                       ID(contestCtx.Contest.ID),
		Name:                     contestCtx.Contest.Title,
		StartTime:                Time(contestStart),
		Duration:                 RelTime(config.Duration),
		ScoreboardFreezeDuration: RelTime(config.Duration - config.FreezeBeginDuration),
		ScoreboardType:           "pass-fail",
		PenaltyTime:              20,
	})
	if contestCtx.Stage != managers.ContestNotPlanned &&
		contestCtx.Stage != managers.ContestNotStarted {
		events = append(events, State{
			Started: Time(contestStart),
		})
	}
	contestProblems, err := v.core.ContestProblems.FindByContest(contestCtx.Contest.ID)
	if err != nil {
		return err
	}
	sortFunc(contestProblems, func(lhs, rhs models.ContestProblem) bool {
		return lhs.Code < rhs.Code
	})
	for i, contestProblem := range contestProblems {
		problem, err := v.core.Problems.Get(
			c.Request().Context(), contestProblem.ProblemID,
		)
		if err != nil {
			return err
		}
		events = append(events, Problem{
			ID:      ID(contestProblem.ID),
			Label:   contestProblem.Code,
			Name:    problem.Title,
			Ordinal: i + 1,
		})
	}
	participants, err := v.core.ContestParticipants.FindByContest(contestCtx.Contest.ID)
	if err != nil {
		return err
	}
	contestSolutions, err := v.core.ContestSolutions.FindByContest(contestCtx.Contest.ID)
	if err != nil {
		return err
	}
	solutionsByParticipant := map[int64][]models.ContestSolution{}
	for _, solution := range contestSolutions {
		solutionsByParticipant[solution.ParticipantID] = append(
			solutionsByParticipant[solution.ParticipantID], solution,
		)
	}
	closable := contestCtx.Stage == managers.ContestFinished
	for _, participant := range participants {
		if participant.Kind != models.RegularParticipant {
			continue
		}
		events = append(events, Team{
			ID:          ID(participant.ID),
			Name:        "Test participant",
			DisplayName: "Test participant",
		})
		beginTime := int64(config.BeginTime)
		if participant.Kind == models.RegularParticipant {
			var participantConfig models.RegularParticipantConfig
			if err := participant.ScanConfig(&participantConfig); err != nil {
				continue
			}
			if participantConfig.BeginTime != 0 {
				beginTime = int64(participantConfig.BeginTime)
			}
		}
		participantSolutions, ok := solutionsByParticipant[participant.ID]
		if !ok {
			continue
		}
		for _, contestSolution := range participantSolutions {
			solution, err := v.core.Solutions.Get(
				c.Request().Context(), contestSolution.ContestID,
			)
			if err != nil {
				closable = false
				continue
			}
			contestTime := solution.CreateTime - beginTime
			if contestTime < 0 {
				contestTime = 1
			}
			events = append(events, Submission{
				ID:          ID(solution.ID),
				TeamID:      ID(contestSolution.ParticipantID),
				ProblemID:   ID(contestSolution.ProblemID),
				LanguageID:  ID(solution.CompilerID),
				ContestTime: RelTime(contestTime),
			})
			report, err := solution.GetReport()
			if err != nil || report.Verdict == models.Failed {
				closable = false
				continue
			}
			events = append(events, Judgement{
				ID:               ID(solution.ID),
				SubmissionID:     ID(solution.ID),
				StartContestTime: RelTime(contestTime),
			})
		}
	}
	if closable {
		endedValue := Time(contestFinish)
		state := State{
			Started: Time(contestStart),
			Ended:   &endedValue,
		}
		if config.FreezeBeginDuration != 0 {
			frozenValue := Time(contestStart.Add(time.Second * time.Duration(config.FreezeBeginDuration)))
			state.Frozen = &frozenValue
		}
		state.Finalized = state.Ended
		state.EndOfUpdates = state.Ended
		events = append(events, state)
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
	if closable {
		return nil
	}
	// TODO: Support of realtime updates.
	return nil
}

type Event struct {
	Type  string    `json:"type"`
	ID    ID        `json:"id,omitempty"`
	Data  EventData `json:"data"`
	Token string    `json:"token"`
}

type EventData interface {
	Kind() string
	ObjectID() ID
}

type ID int64

func (v ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprint(v))
}

type Time time.Time

func (v Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(v).Format("2006-01-02T15:04:05-07:00"))
}

type RelTime int

func (v RelTime) MarshalJSON() ([]byte, error) {
	hours := v / 3600
	minutes := (v / 60) % 60
	seconds := v % 60
	return json.Marshal(fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds))
}

type Contest struct {
	ID                       ID      `json:"id"`
	Name                     string  `json:"name"`
	StartTime                Time    `json:"start_time"`
	Duration                 RelTime `json:"duration"`
	ScoreboardFreezeDuration RelTime `json:"scoreboard_freeze_duration"`
	ScoreboardType           string  `json:"scoreboard_type"`
	PenaltyTime              int     `json:"penalty_time"`
}

func (e Contest) Kind() string {
	return "contest"
}

func (e Contest) ObjectID() ID {
	return e.ID
}

type Problem struct {
	ID            ID     `json:"id"`
	Label         string `json:"label"`
	Name          string `json:"name"`
	Ordinal       int    `json:"ordinal"`
	TimeLimit     int    `json:"time_limit"`
	TestDataCount int    `json:"test_data_count"`
}

func (e Problem) Kind() string {
	return "problems"
}

func (e Problem) ObjectID() ID {
	return e.ID
}

type Team struct {
	ID          ID     `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

func (e Team) Kind() string {
	return "teams"
}

func (e Team) ObjectID() ID {
	return e.ID
}

type Submission struct {
	ID          ID      `json:"id"`
	TeamID      ID      `json:"team_id"`
	ProblemID   ID      `json:"problem_id"`
	LanguageID  ID      `json:"language_id"`
	ContestTime RelTime `json:"contest_time"`
}

func (e Submission) Kind() string {
	return "submissions"
}

func (e Submission) ObjectID() ID {
	return e.ID
}

type Judgement struct {
	ID               ID      `json:"id"`
	SubmissionID     ID      `json:"submission_id"`
	StartContestTime RelTime `json:"start_contest_time"`
}

func (e Judgement) Kind() string {
	return "judgements"
}

func (e Judgement) ObjectID() ID {
	return e.ID
}

type State struct {
	Started      Time  `json:"started"`
	Ended        *Time `json:"ended,omitempty"`
	Frozen       *Time `json:"frozen,omitempty"`
	Finalized    *Time `json:"finalized,omitempty"`
	EndOfUpdates *Time `json:"end_of_updates,omitempty"`
}

func (e State) Kind() string {
	return "state"
}

func (e State) ObjectID() ID {
	return 0
}

func sortFunc[T any](a []T, less func(T, T) bool) {
	impl := sortFuncImpl[T]{data: a, less: less}
	sort.Sort(&impl)
}

type sortFuncImpl[T any] struct {
	data []T
	less func(T, T) bool
}

func (s *sortFuncImpl[T]) Len() int {
	return len(s.data)
}

func (s *sortFuncImpl[T]) Swap(i, j int) {
	s.data[i], s.data[j] = s.data[j], s.data[i]
}

func (s *sortFuncImpl[T]) Less(i, j int) bool {
	return s.less(s.data[i], s.data[j])
}

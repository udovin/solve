package ccs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
)

func (v *View) getEventFeed(c echo.Context) error {
	solutionEvents := v.core.Solutions.Events()
	lastSolutionEventID, err := solutionEvents.LastEventID(c.Request().Context())
	if err != nil {
		return err
	}
	if err := v.core.Solutions.Sync(c.Request().Context()); err != nil {
		return err
	}
	solutionsConsumer := db.NewEventConsumer[models.SolutionEvent](solutionEvents, lastSolutionEventID)
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	if !contestCtx.HasPermission(perms.ObserveContestFullStandingsRole) {
		return c.NoContent(http.StatusForbidden)
	}
	config := contestCtx.ContestConfig
	events := []EventData{}
	events = append(events, getContest(contestCtx))
	events = append(events, getJudgementTypes()...)
	if err := func() error {
		compilers, err := v.core.Compilers.All(c.Request().Context(), 0)
		if err != nil {
			return err
		}
		defer func() { _ = compilers.Close() }()
		for compilers.Next() {
			compiler := compilers.Row()
			config, err := compiler.GetConfig()
			if err != nil {
				return err
			}
			events = append(events, Language{
				ID:         ID(compiler.ID),
				Name:       compiler.Name,
				Extensions: config.Extensions,
			})
		}
		return compilers.Err()
	}(); err != nil {
		return fmt.Errorf("cannot get compilers: %w", err)
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
			return fmt.Errorf("cannot get problem %d: %w", contestProblem.ProblemID, err)
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
		return fmt.Errorf("cannot get participants: %w", err)
	}
	solutionsByParticipant := map[int64][]models.ContestSolution{}
	if err := func() error {
		contestSolutions, err := v.core.ContestSolutions.FindByContest(
			c.Request().Context(), contestCtx.Contest.ID,
		)
		if err != nil {
			return fmt.Errorf("cannot get solutions: %w", err)
		}
		defer func() { _ = contestSolutions.Close() }()
		for contestSolutions.Next() {
			solution := contestSolutions.Row()
			solutionsByParticipant[solution.ParticipantID] = append(
				solutionsByParticipant[solution.ParticipantID], solution,
			)
		}
		return nil
	}(); err != nil {
		return err
	}
	events = append(events, getContestState(contestCtx))
	runningSolutions := map[int64]struct{}{}
	for _, participant := range participants {
		if participant.Kind != models.RegularParticipant {
			continue
		}
		accountInfo, err := v.getAccountInfo(c.Request().Context(), participant.AccountID)
		if err != nil {
			return fmt.Errorf("cannot get account %d: %w", participant.AccountID, err)
		}
		if accountInfo == nil {
			continue
		}
		events = append(events, Team{
			ID:          ID(participant.ID),
			Name:        accountInfo.Title,
			DisplayName: accountInfo.Title,
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
				c.Request().Context(), contestSolution.SolutionID,
			)
			if err != nil {
				return fmt.Errorf("cannot get solution %d: %w", contestSolution.SolutionID, err)
			}
			realTime := time.Unix(solution.CreateTime, 0)
			contestTime := solution.CreateTime - beginTime
			if contestTime < 0 {
				realTime = time.Unix(beginTime, 0)
				contestTime = 0
			}
			events = append(events, Submission{
				ID:          ID(solution.ID),
				TeamID:      ID(contestSolution.ParticipantID),
				ProblemID:   ID(contestSolution.ProblemID),
				LanguageID:  ID(solution.CompilerID),
				Time:        Time(realTime),
				ContestTime: RelTime(contestTime),
			})
			runningSolutions[solution.ID] = struct{}{}
			report, err := solution.GetReport()
			if err != nil {
				continue
			}
			if report == nil || report.Verdict == models.Failed {
				continue
			}
			events = append(events, Judgement{
				ID:               ID(solution.ID),
				SubmissionID:     ID(solution.ID),
				JudgementTypeID:  getJudgementID(report.Verdict),
				StartTime:        Time(realTime),
				StartContestTime: RelTime(contestTime),
				EndTime:          Time(realTime),
				EndContestTime:   RelTime(contestTime),
			})
			delete(runningSolutions, solution.ID)
		}
	}
	c.Response().WriteHeader(http.StatusOK)
	flushEvents := func() error {
		for _, eventData := range events {
			event := Event{
				Type:  eventData.Kind(),
				Data:  eventData,
				Token: eventData.Token(),
			}
			if id := eventData.ObjectID(); id != nil {
				event.ID = fmt.Sprint(id)
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
		events = nil
		c.Response().Flush()
		return nil
	}
	if contestCtx.Stage == managers.ContestFinished && len(runningSolutions) == 0 {
		state := getContestState(contestCtx)
		state.Finalized = state.Ended
		state.EndOfUpdates = state.Ended
		events = append(events, state)
		return flushEvents()
	}
	if err := flushEvents(); err != nil {
		return err
	}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case <-ticker.C:
		}
		contest, err := v.core.Contests.Get(c.Request().Context(), contestCtx.Contest.ID)
		if err != nil {
			return fmt.Errorf("cannot update contest: %w", err)
		}
		contestCtx, err = v.contests.BuildContext(contestCtx.AccountContext, contest)
		if err != nil {
			return fmt.Errorf("cannot build contest context: %w", err)
		}
		config := contestCtx.ContestConfig
		events = append(events, getContest(contestCtx), getContestState(contestCtx))
		if err := solutionsConsumer.ConsumeEvents(c.Request().Context(), func(event models.SolutionEvent) error {
			solution := event.Solution
			contestSolution, err := v.core.ContestSolutions.FindOne(c.Request().Context(), db.FindQuery{
				Where: gosql.Column("solution_id").Equal(solution.ID),
			})
			if err != nil {
				return fmt.Errorf("cannot find contest solution %d: %w", solution.ID, err)
			}
			if contestSolution.ContestID != contestCtx.Contest.ID {
				return nil
			}
			participant, err := v.core.ContestParticipants.Get(c.Request().Context(), contestSolution.ParticipantID)
			if err != nil {
				return fmt.Errorf("cannot get participant %d: %w", contestSolution.ParticipantID, err)
			}
			if participant.Kind != models.RegularParticipant {
				return nil
			}
			beginTime := int64(config.BeginTime)
			if participant.Kind == models.RegularParticipant {
				var participantConfig models.RegularParticipantConfig
				if err := participant.ScanConfig(&participantConfig); err != nil {
					return err
				}
				if participantConfig.BeginTime != 0 {
					beginTime = int64(participantConfig.BeginTime)
				}
			}
			realTime := time.Unix(solution.CreateTime, 0)
			contestTime := solution.CreateTime - beginTime
			if contestTime < 0 {
				realTime = time.Unix(beginTime, 0)
				contestTime = 0
			}
			events = append(events, Submission{
				ID:          ID(solution.ID),
				TeamID:      ID(contestSolution.ParticipantID),
				ProblemID:   ID(contestSolution.ProblemID),
				LanguageID:  ID(solution.CompilerID),
				Time:        Time(realTime),
				ContestTime: RelTime(contestTime),
			})
			runningSolutions[solution.ID] = struct{}{}
			report, err := solution.GetReport()
			if err != nil {
				return err
			}
			if report == nil || report.Verdict == models.Failed {
				return nil
			}
			events = append(events, Judgement{
				ID:               ID(solution.ID),
				SubmissionID:     ID(solution.ID),
				JudgementTypeID:  getJudgementID(report.Verdict),
				StartTime:        Time(realTime),
				StartContestTime: RelTime(contestTime),
				EndTime:          Time(realTime),
				EndContestTime:   RelTime(contestTime),
			})
			delete(runningSolutions, solution.ID)
			return nil
		}); err != nil {
			c.Logger().Error(err)
		}
		if err := flushEvents(); err != nil {
			return err
		}
		if contestCtx.Stage == managers.ContestFinished && len(runningSolutions) == 0 {
			state := getContestState(contestCtx)
			state.Finalized = state.Ended
			state.EndOfUpdates = state.Ended
			events = append(events, state)
			return flushEvents()
		}
	}
}

func getJudgementID(verdict models.Verdict) string {
	switch verdict {
	case models.Accepted:
		return "AC"
	case models.Rejected:
		return "RJ"
	case models.CompilationError:
		return "CE"
	case models.Failed:
		return "FL"
	case models.MemoryLimitExceeded:
		return "MLE"
	case models.TimeLimitExceeded:
		return "TLE"
	case models.RuntimeError:
		return "RE"
	case models.WrongAnswer:
		return "WA"
	case models.PresentationError:
		return "PE"
	default:
		return ""
	}
}

func getJudgementTypes() []EventData {
	return []EventData{
		JudgementType{ID: getJudgementID(models.Accepted), Name: "Accepted", Solved: true},
		JudgementType{ID: getJudgementID(models.Rejected), Name: "Rejected", Penalty: true},
		JudgementType{ID: getJudgementID(models.CompilationError), Name: "Compilation error"},
		JudgementType{ID: getJudgementID(models.Failed), Name: "Failed"},
		JudgementType{ID: getJudgementID(models.MemoryLimitExceeded), Name: "Memory limit exceeded", Penalty: true},
		JudgementType{ID: getJudgementID(models.TimeLimitExceeded), Name: "Time limit exceeded", Penalty: true},
		JudgementType{ID: getJudgementID(models.RuntimeError), Name: "Runtime error", Penalty: true},
		JudgementType{ID: getJudgementID(models.WrongAnswer), Name: "Wrong answer", Penalty: true},
		JudgementType{ID: getJudgementID(models.PresentationError), Name: "Presentation error", Penalty: true},
	}
}

func getContest(contestCtx *managers.ContestContext) Contest {
	config := contestCtx.ContestConfig
	contestStart := time.Unix(int64(config.BeginTime), 0)
	freezeDuration := 0
	if config.FreezeBeginDuration > 0 {
		freezeDuration = config.Duration - config.FreezeBeginDuration
	}
	return Contest{
		ID:                       ID(contestCtx.Contest.ID),
		Name:                     contestCtx.Contest.Title,
		StartTime:                Time(contestStart),
		Duration:                 RelTime(config.Duration),
		ScoreboardFreezeDuration: RelTime(freezeDuration),
		ScoreboardType:           "pass-fail",
		PenaltyTime:              20,
	}
}

func getContestState(contestCtx *managers.ContestContext) State {
	config := contestCtx.ContestConfig
	contestStart := time.Unix(int64(config.BeginTime), 0)
	contestFinish := contestStart.Add(time.Second * time.Duration(config.Duration))
	state := State{}
	if contestCtx.Stage >= managers.ContestStarted {
		startedValue := Time(contestStart)
		state.Started = &startedValue
		if config.FreezeBeginDuration != 0 {
			contestFreeze := contestStart.Add(time.Second * time.Duration(config.FreezeBeginDuration))
			if !contestCtx.Now.Before(contestFreeze) {
				frozenValue := Time(contestFreeze)
				state.Frozen = &frozenValue
			}
		}
	}
	if contestCtx.Stage >= managers.ContestFinished {
		endedValue := Time(contestFinish)
		state.Ended = &endedValue
	}
	return state
}

type accountInfo struct {
	Title string
}

func (v *View) getAccountInfo(
	ctx context.Context, accountID int64,
) (*accountInfo, error) {
	info := accountInfo{}
	account, err := v.core.Accounts.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	switch account.Kind {
	case models.UserAccount:
		user, err := v.core.Users.GetByAccount(account.ID)
		if err != nil {
			return nil, fmt.Errorf("cannot get user: %w", err)
		}
		info.Title = user.Login
	case models.ScopeUserAccount:
		user, err := v.core.ScopeUsers.GetByAccount(account.ID)
		if err != nil {
			return nil, fmt.Errorf("cannot get scope user: %w", err)
		}
		info.Title = string(user.Title)
	default:
		return nil, nil
	}
	return &info, nil
}

type Event struct {
	Type  string    `json:"type"`
	ID    string    `json:"id,omitempty"`
	Data  EventData `json:"data"`
	Token string    `json:"token"`
}

type EventData interface {
	Kind() string
	ObjectID() any
	Token() string
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

func (e Contest) ObjectID() any {
	return nil
}

func (e Contest) Token() string {
	return fmt.Sprint(time.Now().UnixNano())
}

type JudgementType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Penalty bool   `json:"penalty"`
	Solved  bool   `json:"solved"`
}

func (e JudgementType) Kind() string {
	return "judgement-types"
}

func (e JudgementType) ObjectID() any {
	return e.ID
}

func (e JudgementType) Token() string {
	return fmt.Sprint(time.Now().UnixNano())
}

type Language struct {
	ID                 ID       `json:"id"`
	Name               string   `json:"name"`
	EntryPointRequired bool     `json:"entry_point_required"`
	Extensions         []string `json:"extensions"`
}

func (e Language) Kind() string {
	return "languages"
}

func (e Language) ObjectID() any {
	return e.ID
}

func (e Language) Token() string {
	return fmt.Sprint(time.Now().UnixNano())
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

func (e Problem) ObjectID() any {
	return e.ID
}

func (e Problem) Token() string {
	return fmt.Sprint(time.Now().UnixNano())
}

type Organization struct {
	ID   ID     `json:"id"`
	Name string `json:"name"`
}

func (e Organization) Kind() string {
	return "organizations"
}

func (e Organization) ObjectID() any {
	return e.ID
}

func (e Organization) Token() string {
	return fmt.Sprint(time.Now().UnixNano())
}

type Team struct {
	ID             ID     `json:"id"`
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	OrganizationID ID     `json:"organization_id"`
}

func (e Team) Kind() string {
	return "teams"
}

func (e Team) ObjectID() any {
	return e.ID
}

func (e Team) Token() string {
	return fmt.Sprint(time.Now().UnixNano())
}

type Submission struct {
	ID          ID      `json:"id"`
	TeamID      ID      `json:"team_id"`
	ProblemID   ID      `json:"problem_id"`
	LanguageID  ID      `json:"language_id"`
	Time        Time    `json:"time"`
	ContestTime RelTime `json:"contest_time"`
}

func (e Submission) Kind() string {
	return "submissions"
}

func (e Submission) ObjectID() any {
	return e.ID
}

func (e Submission) Token() string {
	return fmt.Sprint(e.ID)
}

type Judgement struct {
	ID               ID      `json:"id"`
	SubmissionID     ID      `json:"submission_id"`
	JudgementTypeID  string  `json:"judgement_type_id,omitempty"`
	StartTime        Time    `json:"start_time"`
	StartContestTime RelTime `json:"start_contest_time"`
	EndTime          Time    `json:"end_time"`
	EndContestTime   RelTime `json:"end_contest_time"`
}

func (e Judgement) Kind() string {
	return "judgements"
}

func (e Judgement) ObjectID() any {
	return e.ID
}

func (e Judgement) Token() string {
	return fmt.Sprint(time.Now().UnixNano())
}

type State struct {
	Started      *Time `json:"started,omitempty"`
	Ended        *Time `json:"ended,omitempty"`
	Frozen       *Time `json:"frozen,omitempty"`
	Finalized    *Time `json:"finalized,omitempty"`
	EndOfUpdates *Time `json:"end_of_updates,omitempty"`
}

func (e State) Kind() string {
	return "state"
}

func (e State) ObjectID() any {
	return nil
}

func (e State) Token() string {
	return fmt.Sprint(time.Now().UnixNano())
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

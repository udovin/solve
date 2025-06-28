package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/udovin/solve/api/schema"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
)

func (v *View) registerContestFakeHandlers(g *echo.Group) {
	g.POST(
		"/v0/contests/:contest/fake-participants", v.createContestFakeParticipant,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(perms.CreateContestParticipantRole),
	)
	g.DELETE(
		"/v0/contests/:contest/fake-participants/:participant", v.deleteContestFakeParticipant,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(perms.DeleteContestParticipantRole),
	)
	g.POST(
		"/v0/contests/:contest/fake-solutions", v.createContestFakeSolution,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(perms.CreateContestSolutionRole),
	)
	g.DELETE(
		"/v0/contests/:contest/fake-solutions/:solution", v.deleteContestFakeSolution,
		v.extractAuth(v.sessionAuth), v.extractContest,
		v.requirePermission(perms.DeleteContestSolutionRole),
	)
}

type createContestFakeParticipantForm struct {
	ExternalID string `json:"external_id"`
	Title      string `json:"title"`
}

func (f *createContestFakeParticipantForm) Update(
	c echo.Context, o *models.ContestFakeParticipant,
) error {
	errors := errorFields{}
	if len(f.ExternalID) > 32 {
		errors["external_id"] = errorField{
			Message: localize(c, "External ID is too long."),
		}
	}
	title := []rune(f.Title)
	if len(title) < 4 {
		errors["title"] = errorField{
			Message: localize(c, "Title is too short."),
		}
	} else if len(title) > 64 {
		errors["title"] = errorField{
			Message: localize(c, "Title is too long."),
		}
	}
	if len(errors) > 0 {
		return errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	if len(f.ExternalID) > 0 {
		o.ExternalID = NString(f.ExternalID)
	}
	o.Title = f.Title
	return nil
}

func (v *View) createContestFakeParticipant(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	var form createContestFakeParticipantForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var participant models.ContestFakeParticipant
	if err := form.Update(c, &participant); err != nil {
		return err
	}
	participant.ContestID = contest.ID
	if err := v.core.ContestFakeParticipants.Create(getContext(c), &participant); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, makeContestFakeParticipant(participant))
}

func (v *View) deleteContestFakeParticipant(c echo.Context) error {
	return nil
}

type createContestFakeSolutionForm struct {
	ExternalID            string         `json:"external_id"`
	ParticipantExternalID string         `json:"participant_external_id"`
	ParticipantID         int64          `json:"participant_id"`
	ProblemCode           string         `json:"problem_code"`
	Verdict               models.Verdict `json:"verdict"`
	Points                *float64       `json:"points"`
	ContestTime           int64          `json:"contest_time"`
}

func (f *createContestFakeSolutionForm) Update(
	c echo.Context, o *models.ContestFakeSolution,
) error {
	errors := errorFields{}
	if len(f.ExternalID) > 32 {
		errors["external_id"] = errorField{
			Message: localize(c, "External ID is too long."),
		}
	}
	if len(errors) > 0 {
		return errorResponse{
			Code:          http.StatusBadRequest,
			Message:       localize(c, "Form has invalid fields."),
			InvalidFields: errors,
		}
	}
	if len(f.ExternalID) > 0 {
		o.ExternalID = NString(f.ExternalID)
	}
	o.ContestTime = f.ContestTime
	report := models.FakeSolutionReport{
		Verdict: f.Verdict,
		Points:  f.Points,
	}
	if err := o.SetReport(&report); err != nil {
		return fmt.Errorf("cannot set report")
	}
	return nil
}

func (v *View) createContestFakeSolution(c echo.Context) error {
	contestCtx, ok := c.Get(contestCtxKey).(*managers.ContestContext)
	if !ok {
		return fmt.Errorf("contest not extracted")
	}
	contest := contestCtx.Contest
	var form createContestFakeSolutionForm
	if err := c.Bind(&form); err != nil {
		c.Logger().Warn(err)
		return c.NoContent(http.StatusBadRequest)
	}
	var solution models.ContestFakeSolution
	if err := form.Update(c, &solution); err != nil {
		return err
	}
	problem, err := findContestProblem(getContext(c), v.core, contest.ID, form.ProblemCode)
	if err != nil {
		return err
	}
	if problem == nil || problem.ContestID != contest.ID {
		return errorResponse{
			Code: http.StatusNotFound,
			Message: localize(
				c, "Problem {code} does not exists.",
				replaceField("code", form.ProblemCode),
			),
		}
	}
	var participant models.ContestFakeParticipant
	if len(form.ParticipantExternalID) > 0 {
		participant, err = v.core.ContestFakeParticipants.GetByExternalID(
			getContext(c), contest.ID, form.ParticipantExternalID,
		)
	} else {
		participant, err = v.core.ContestFakeParticipants.Get(getContext(c), form.ParticipantID)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return errorResponse{
				Code:    http.StatusNotFound,
				Message: localize(c, "Participant not found."),
			}
		}
		return err
	}
	if participant.ContestID != contest.ID {
		return errorResponse{
			Code:    http.StatusNotFound,
			Message: localize(c, "Participant not found."),
		}
	}
	solution.ContestID = contest.ID
	solution.ProblemID = problem.ID
	solution.ParticipantID = participant.ID
	if err := v.core.ContestFakeSolutions.Create(getContext(c), &solution); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, v.makeContestFakeSolution(c, solution))
}

func (v *View) deleteContestFakeSolution(c echo.Context) error {
	return nil
}

func makeContestFakeParticipant(participant models.ContestFakeParticipant) schema.ContestFakeParticipant {
	return schema.ContestFakeParticipant{
		ID:         participant.ID,
		ExternalID: string(participant.ExternalID),
		Title:      participant.Title,
	}
}

type ContestFakeSolution struct {
	ID          int64                          `json:"id"`
	ExternalID  string                         `json:"external_id,omitempty"`
	Problem     *ContestProblem                `json:"problem,omitempty"`
	Participant *schema.ContestFakeParticipant `json:"participant,omitempty"`
	Report      *SolutionReport                `json:"report"`
	ContestTime int64                          `json:"contest_time"`
}

func (v *View) makeContestFakeSolution(c echo.Context, solution models.ContestFakeSolution) ContestFakeSolution {
	resp := ContestFakeSolution{
		ID:          solution.ID,
		ExternalID:  string(solution.ExternalID),
		ContestTime: solution.ContestTime,
	}
	if problem, err := v.core.ContestProblems.Get(
		getContext(c), solution.ProblemID,
	); err == nil {
		problemResp := v.makeContestProblem(c, problem, false)
		resp.Problem = &problemResp
	}
	if participant, err := v.core.ContestFakeParticipants.Get(
		getContext(c), solution.ParticipantID,
	); err == nil {
		participantResp := makeContestFakeParticipant(participant)
		resp.Participant = &participantResp
	}
	report, err := solution.GetReport()
	if err == nil {
		reportResp := SolutionReport{}
		reportResp.Verdict = report.Verdict.String()
		reportResp.Points = report.Points
		resp.Report = &reportResp
	}
	return resp
}

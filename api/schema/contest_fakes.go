package schema

import "github.com/udovin/solve/internal/models"

type ContestFakeParticipant struct {
	ID         int64  `json:"id"`
	ExternalID string `json:"external_id,omitempty"`
	Title      string `json:"title"`
}

type CreateContestFakeParticipantRequest struct {
	ContestID  int64  `json:"-"`
	ExternalID string `json:"external_id"`
	Title      string `json:"title"`
}

type CreateContestFakeParticipantResponse struct {
	ContestFakeParticipant
}

type SolutionVerdict = models.Verdict

type CreateContestFakeSolutionRequest struct {
	ContestID             int64           `json:"-"`
	ExternalID            string          `json:"external_id"`
	ParticipantExternalID string          `json:"participant_external_id,omitempty"`
	ParticipantID         int64           `json:"participant_id,omitempty"`
	ProblemCode           string          `json:"problem_code"`
	Verdict               SolutionVerdict `json:"verdict"`
	Points                *float64        `json:"points"`
	ContestTime           int64           `json:"contest_time"`
}

type CreateContestFakeSolutionResponse struct{}

package api

import (
	"context"

	"github.com/udovin/solve/api/schema"
)

type ContestFakesClient interface {
	CreateContestFakeParticipant(ctx context.Context, r schema.CreateContestFakeParticipantRequest) (schema.CreateContestFakeParticipantResponse, error)
	CreateContestFakeSolution(ctx context.Context, r schema.CreateContestFakeSolutionRequest) (schema.CreateContestFakeSolutionResponse, error)
}

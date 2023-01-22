package managers

import (
	"database/sql"
	"fmt"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

type ContestStandingsManager struct {
	contestParticipants *models.ContestParticipantStore
	contestSolutions    *models.ContestSolutionStore
	solutions           *models.SolutionStore
}

func NewContestStandingsManager(core *core.Core) *ContestStandingsManager {
	return &ContestStandingsManager{
		contestParticipants: core.ContestParticipants,
		contestSolutions:    core.ContestSolutions,
		solutions:           core.Solutions,
	}
}

func (m *ContestStandingsManager) BuildStandings(ctx *ContestContext) (*ContestStandings, error) {
	effectiveParticipant := ctx.GetEffectiveParticipant()
	contestSolutions, err := m.contestSolutions.FindByContest(ctx.Contest.ID)
	if err != nil {
		return nil, err
	}
	participants, err := m.contestParticipants.FindByContest(ctx.Contest.ID)
	if err != nil {
		return nil, err
	}
	participantPos := map[int64]int{}
	for i, participant := range participants {
		participantPos[participant.ID] = i
	}
	for _, contestSolution := range contestSolutions {
		pos, ok := participantPos[contestSolution.ParticipantID]
		if !ok {
			continue
		}
		participant := participants[pos]
		solution, err := m.solutions.Get(contestSolution.SolutionID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, fmt.Errorf("cannot get solution: %w", err)
		}
		_ = participant
		_ = solution
	}
	_ = effectiveParticipant
	return nil, fmt.Errorf("not implemented")
}

type ContestStandings struct {
}

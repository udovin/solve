package managers

import (
	"context"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

type ContestManager struct {
	Contests     *models.ContestStore
	Participants *models.ContestParticipantStore
}

func NewContestManager(core *core.Core) *ContestManager {
	return &ContestManager{
		Contests:     core.Contests,
		Participants: core.ContestParticipants,
	}
}

func (m *ContestManager) BuildContext(ctx *AccountContext, contest models.Contest) (*ContestContext, error) {
	c := ContestContext{
		AccountContext: ctx,
		Contest:        contest,
		Permissions:    PermissionSet{},
	}
	if account := ctx.Account; account != nil {
		if contest.OwnerID != 0 && account.ID == int64(contest.OwnerID) {
			c.Permissions[models.ObserveContestRole] = struct{}{}
			c.Permissions[models.UpdateContestRole] = struct{}{}
			c.Permissions[models.DeleteContestRole] = struct{}{}
			c.Permissions[models.ObserveContestProblemsRole] = struct{}{}
			c.Permissions[models.ObserveContestProblemRole] = struct{}{}
			c.Permissions[models.CreateContestProblemRole] = struct{}{}
			c.Permissions[models.DeleteContestProblemRole] = struct{}{}
			c.Permissions[models.ObserveContestParticipantsRole] = struct{}{}
			c.Permissions[models.ObserveContestParticipantRole] = struct{}{}
			c.Permissions[models.CreateContestParticipantRole] = struct{}{}
			c.Permissions[models.DeleteContestParticipantRole] = struct{}{}
			c.Permissions[models.ObserveContestSolutionsRole] = struct{}{}
			c.Permissions[models.ObserveContestSolutionRole] = struct{}{}
			c.Permissions[models.CreateContestSolutionRole] = struct{}{}
			c.Permissions[models.UpdateContestSolutionRole] = struct{}{}
			c.Permissions[models.DeleteContestSolutionRole] = struct{}{}
			c.Permissions[models.ObserveContestRole] = struct{}{}
			c.Permissions[models.ObserveContestRole] = struct{}{}
			c.Permissions[models.ObserveContestRole] = struct{}{}
		}
		participants, err := m.Participants.FindByContestAccount(contest.ID, ctx.Account.ID)
		if err != nil {
			return nil, err
		}
		c.Participants = participants
	}
	return &c, nil
}

type ContestContext struct {
	*AccountContext
	Contest      models.Contest
	Participants []models.ContestParticipant
	Permissions  PermissionSet
}

func (c *ContestContext) HasPermission(name string) bool {
	return c.Permissions.HasPermission(name) || c.AccountContext.HasPermission(name)
}

var (
	_ context.Context = (*ContestContext)(nil)
	_ Permissions     = (*ContestContext)(nil)
)

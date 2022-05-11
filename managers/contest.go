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

func addContestManagerPermissions(permissions PermissionSet) {
	permissions.AddPermission(
		models.ObserveContestRole,
		models.UpdateContestRole,
		models.ObserveContestProblemsRole,
		models.ObserveContestProblemRole,
		models.CreateContestProblemRole,
		models.DeleteContestProblemRole,
		models.ObserveContestParticipantsRole,
		models.ObserveContestParticipantRole,
		models.CreateContestParticipantRole,
		models.DeleteContestParticipantRole,
		models.ObserveContestSolutionsRole,
		models.ObserveContestSolutionRole,
		models.CreateContestSolutionRole,
		models.UpdateContestSolutionRole,
		models.DeleteContestSolutionRole,
	)
}

func (m *ContestManager) BuildContext(ctx *AccountContext, contest models.Contest) (*ContestContext, error) {
	c := ContestContext{
		AccountContext: ctx,
		Contest:        contest,
		Permissions:    PermissionSet{},
	}
	if account := ctx.Account; account != nil {
		if contest.OwnerID != 0 && account.ID == int64(contest.OwnerID) {
			addContestManagerPermissions(c.Permissions)
			c.Permissions.AddPermission(models.DeleteContestRole)
		}
		participants, err := m.Participants.FindByContestAccount(contest.ID, ctx.Account.ID)
		if err != nil {
			return nil, err
		}
		for _, participant := range participants {
			switch participant.Kind {
			case models.RegularParticipant:
				c.Permissions.AddPermission(models.ObserveContestRole)
			case models.UpsolvingParticipant:
				c.Permissions.AddPermission(models.ObserveContestRole)
			case models.VirtualParticipant:
				c.Permissions.AddPermission(models.ObserveContestRole)
			case models.ManagerParticipant:
				addContestManagerPermissions(c.Permissions)
			}
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

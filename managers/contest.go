package managers

import (
	"context"
	"fmt"

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

func addContestRegularPermissions(permissions PermissionSet) {
	permissions.AddPermission(
		models.ObserveContestRole,
	)
}

func addContestVirtualPermissions(permissions PermissionSet) {
	permissions.AddPermission(
		models.ObserveContestRole,
	)
}

func addContestUpsolvingPermissions(permissions PermissionSet) {
	permissions.AddPermission(
		models.ObserveContestRole,
	)
}

func (m *ContestManager) BuildContext(ctx *AccountContext, contest models.Contest) (*ContestContext, error) {
	config, err := contest.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to build contest context: %w", err)
	}
	c := ContestContext{
		AccountContext: ctx,
		Contest:        contest,
		Permissions:    PermissionSet{},
	}
	now := models.GetNow(ctx).Unix()
	if account := ctx.Account; account != nil {
		if contest.OwnerID != 0 && account.ID == int64(contest.OwnerID) {
			addContestManagerPermissions(c.Permissions)
			c.Permissions.AddPermission(models.DeleteContestRole)
		}
		participants, err := m.Participants.FindByContestAccount(contest.ID, account.ID)
		if err != nil {
			return nil, fmt.Errorf("unable to build contest context: %w", err)
		}
		hasRegular := false
		hasUpsolving := false
		for _, participant := range participants {
			switch participant.Kind {
			case models.RegularParticipant:
				hasRegular = true
				addContestRegularPermissions(c.Permissions)
			case models.UpsolvingParticipant:
				hasUpsolving = true
				addContestUpsolvingPermissions(c.Permissions)
			case models.VirtualParticipant:
				addContestVirtualPermissions(c.Permissions)
			case models.ManagerParticipant:
				addContestManagerPermissions(c.Permissions)
			}
		}
		c.Participants = participants
		if config.BeginTime != 0 {
			beginTime := int64(config.BeginTime)
			endTime := beginTime + int64(config.Duration)
			if !hasRegular && config.EnableRegistration &&
				now < beginTime {
				c.Permissions.AddPermission(models.ObserveContestRole)
				c.Permissions.AddPermission(models.RegisterContestRole)
			}
			if !hasUpsolving && config.EnableUpsolving &&
				now > endTime && (hasRegular || config.EnableRegistration) {
				// Add virtual participant for upsolving.
				c.Participants = append(c.Participants, models.ContestParticipant{
					Kind:      models.UpsolvingParticipant,
					ContestID: contest.ID,
					AccountID: account.ID,
				})
				addContestUpsolvingPermissions(c.Permissions)
			}
		}
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

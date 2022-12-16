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
		models.SubmitContestSolutionRole,
	)
}

func addContestRegularPermissions(
	permissions PermissionSet, stage ContestStage,
) {
	permissions.AddPermission(models.ObserveContestRole)
	switch stage {
	case ContestNotStarted:
		permissions.AddPermission(models.DeregisterContestRole)
	case ContestStarted:
		permissions.AddPermission(
			models.ObserveContestProblemsRole,
			models.ObserveContestProblemRole,
			models.ObserveContestSolutionsRole,
			models.SubmitContestSolutionRole,
		)
	case ContestFinished:
		permissions.AddPermission(
			models.ObserveContestProblemsRole,
			models.ObserveContestProblemRole,
			models.ObserveContestSolutionsRole,
		)
	}
}

func addContestVirtualPermissions(permissions PermissionSet) {
	permissions.AddPermission(models.ObserveContestRole)
}

func addContestUpsolvingPermissions(
	permissions PermissionSet, stage ContestStage,
) {
	permissions.AddPermission(models.ObserveContestRole)
	if stage == ContestFinished {
		permissions.AddPermission(
			models.ObserveContestProblemsRole,
			models.ObserveContestProblemRole,
			models.ObserveContestSolutionsRole,
			models.SubmitContestSolutionRole,
		)
	}
}

func getParticipantPermissions(
	contest models.Contest, stage ContestStage,
	participant models.ContestParticipant,
) PermissionSet {
	permissions := PermissionSet{}
	switch participant.Kind {
	case models.RegularParticipant:
		addContestRegularPermissions(permissions, stage)
	case models.UpsolvingParticipant:
		addContestUpsolvingPermissions(permissions, stage)
	case models.VirtualParticipant:
		addContestVirtualPermissions(permissions)
	case models.ManagerParticipant:
		addContestManagerPermissions(permissions)
	}
	return permissions
}

func checkEffectiveParticipant(
	stage ContestStage, participant models.ContestParticipant,
) bool {
	switch participant.Kind {
	case models.RegularParticipant:
		return stage == ContestStarted
	case models.UpsolvingParticipant:
		return stage == ContestFinished
	case models.ManagerParticipant:
		return true
	default:
		return false
	}
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
		Stage:          ContestNotPlanned,
	}
	now := models.GetNow(ctx).Unix()
	if config.BeginTime != 0 {
		c.Stage = ContestNotStarted
		if now >= int64(config.BeginTime) {
			c.Stage = ContestStarted
		}
		if now >= int64(config.BeginTime)+int64(config.Duration) {
			c.Stage = ContestFinished
		}
	}
	if account := ctx.Account; account != nil {
		if contest.OwnerID != 0 && account.ID == int64(contest.OwnerID) {
			c.Permissions.AddPermission(models.DeleteContestRole)
		}
		participants, err := m.Participants.FindByContestAccount(contest.ID, account.ID)
		if err != nil {
			return nil, fmt.Errorf("unable to build contest context: %w", err)
		}
		hasRegular := false
		hasUpsolving := false
		hasManager := false
		for _, participant := range participants {
			for permission := range getParticipantPermissions(
				contest, c.Stage, participant,
			) {
				c.Permissions.AddPermission(permission)
			}
			switch participant.Kind {
			case models.RegularParticipant:
				hasRegular = true
			case models.UpsolvingParticipant:
				hasUpsolving = true
			case models.ManagerParticipant:
				hasManager = true
			}
		}
		c.Participants = participants
		if !hasManager && contest.OwnerID != 0 && account.ID == int64(contest.OwnerID) {
			c.Participants = append(c.Participants, models.ContestParticipant{
				Kind:      models.ManagerParticipant,
				ContestID: contest.ID,
				AccountID: account.ID,
			})
			addContestManagerPermissions(c.Permissions)
		}
		if !hasRegular && c.Stage == ContestNotStarted && config.EnableRegistration {
			c.Permissions.AddPermission(models.ObserveContestRole)
			if c.HasPermission(models.RegisterContestsRole) {
				c.Permissions.AddPermission(models.RegisterContestRole)
			}
		}
		if !hasUpsolving && c.Stage == ContestFinished &&
			config.EnableUpsolving && (hasRegular || config.EnableRegistration) {
			// Add virtual participant for upsolving.
			c.Participants = append(c.Participants, models.ContestParticipant{
				Kind:      models.UpsolvingParticipant,
				ContestID: contest.ID,
				AccountID: account.ID,
			})
			addContestUpsolvingPermissions(c.Permissions, c.Stage)
		}
	}
	c.effectivePos = len(c.Participants)
	for i := 0; i < len(c.Participants); i++ {
		if checkEffectiveParticipant(c.Stage, c.Participants[i]) {
			c.effectivePos = i
			break
		}
	}
	return &c, nil
}

type ContestStage int

const (
	ContestNotPlanned ContestStage = iota
	ContestNotStarted
	ContestStarted
	ContestFinished
)

type ContestContext struct {
	*AccountContext
	Contest      models.Contest
	Participants []models.ContestParticipant
	Permissions  PermissionSet
	Stage        ContestStage
	effectivePos int
}

func (c *ContestContext) HasPermission(name string) bool {
	return c.Permissions.HasPermission(name) || c.AccountContext.HasPermission(name)
}

func (c *ContestContext) GetEffectiveParticipant() *models.ContestParticipant {
	if c.effectivePos >= len(c.Participants) {
		return nil
	}
	return &c.Participants[c.effectivePos]
}

func (c *ContestContext) SetEffectiveParticipant(id int64) {
	for i := range c.Participants {
		if c.Participants[i].ID == id {
			if checkEffectiveParticipant(c.Stage, c.Participants[i]) {
				c.effectivePos = i
			}
			break
		}
	}
}

func (c *ContestContext) GetEffectivePermissions() PermissionSet {
	participant := c.GetEffectiveParticipant()
	if participant == nil {
		return PermissionSet{}
	}
	return getParticipantPermissions(c.Contest, c.Stage, *participant)
}

func (c *ContestContext) HasEffectivePermission(name string) bool {
	return c.GetEffectivePermissions().HasPermission(name)
}

var (
	_ context.Context = (*ContestContext)(nil)
	_ Permissions     = (*ContestContext)(nil)
)

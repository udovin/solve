package managers

import (
	"context"
	"fmt"
	"time"

	"github.com/udovin/solve/internal/core"
	"github.com/udovin/solve/internal/db"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/perms"
)

type ContestManager struct {
	contests     *models.ContestStore
	participants *models.ContestParticipantStore
	settings     *models.SettingStore
}

func NewContestManager(core *core.Core) *ContestManager {
	return &ContestManager{
		contests:     core.Contests,
		participants: core.ContestParticipants,
		settings:     core.Settings,
	}
}

func addContestManagerPermissions(permissions perms.PermissionSet) {
	permissions.AddPermission(
		perms.ObserveContestRole,
		perms.UpdateContestRole,
		perms.ObserveContestProblemsRole,
		perms.ObserveContestProblemRole,
		perms.CreateContestProblemRole,
		perms.UpdateContestProblemRole,
		perms.DeleteContestProblemRole,
		perms.ObserveContestParticipantsRole,
		perms.ObserveContestParticipantRole,
		perms.CreateContestParticipantRole,
		perms.DeleteContestParticipantRole,
		perms.ObserveContestSolutionsRole,
		perms.ObserveContestSolutionRole,
		perms.CreateContestSolutionRole,
		perms.UpdateContestSolutionRole,
		perms.DeleteContestSolutionRole,
		perms.SubmitContestSolutionRole,
		perms.ObserveContestStandingsRole,
		perms.ObserveContestFullStandingsRole,
		perms.ObserveSolutionReportTestNumber,
		perms.ObserveSolutionReportCheckerLogs,
		perms.ObserveContestMessagesRole,
		perms.ObserveContestMessageRole,
		perms.CreateContestMessageRole,
		perms.UpdateContestMessageRole,
		perms.DeleteContestMessageRole,
		perms.SubmitContestQuestionRole,
	)
}

func addContestRegularPermissions(
	permissions perms.PermissionSet, stage ContestStage, config *models.ContestConfig,
) {
	permissions.AddPermission(perms.ObserveContestRole)
	switch stage {
	case ContestNotStarted:
		permissions.AddPermission(perms.DeregisterContestRole)
	case ContestStarted:
		permissions.AddPermission(
			perms.ObserveContestProblemsRole,
			perms.ObserveContestProblemRole,
			perms.ObserveContestSolutionsRole,
			perms.SubmitContestSolutionRole,
			perms.ObserveSolutionReportTestNumber,
			perms.ObserveContestMessagesRole,
			perms.SubmitContestQuestionRole,
		)
		if config.StandingsKind != models.DisabledStandings {
			permissions.AddPermission(perms.ObserveContestStandingsRole)
		}
	case ContestFinished:
		permissions.AddPermission(
			perms.ObserveContestProblemsRole,
			perms.ObserveContestProblemRole,
			perms.ObserveContestSolutionsRole,
			perms.ObserveSolutionReportTestNumber,
			perms.ObserveContestMessagesRole,
		)
		if config.StandingsKind != models.DisabledStandings {
			permissions.AddPermission(perms.ObserveContestStandingsRole)
		}
	}
}

func addContestVirtualPermissions(
	permissions perms.PermissionSet, stage ContestStage, config *models.ContestConfig,
) {
	permissions.AddPermission(perms.ObserveContestRole)
	switch stage {
	case ContestNotStarted:
		permissions.AddPermission(perms.DeregisterContestRole)
	case ContestStarted:
		permissions.AddPermission(
			perms.ObserveContestProblemsRole,
			perms.ObserveContestProblemRole,
			perms.ObserveContestSolutionsRole,
			perms.SubmitContestSolutionRole,
			perms.ObserveSolutionReportTestNumber,
			perms.ObserveContestMessagesRole,
			perms.SubmitContestQuestionRole,
		)
		if config.StandingsKind != models.DisabledStandings {
			permissions.AddPermission(perms.ObserveContestStandingsRole)
		}
	case ContestFinished:
		permissions.AddPermission(
			perms.ObserveContestProblemsRole,
			perms.ObserveContestProblemRole,
			perms.ObserveContestSolutionsRole,
			perms.ObserveSolutionReportTestNumber,
			perms.ObserveContestMessagesRole,
		)
		if config.StandingsKind != models.DisabledStandings {
			permissions.AddPermission(perms.ObserveContestStandingsRole)
		}
	}
}

func addContestUpsolvingPermissions(
	permissions perms.PermissionSet, stage ContestStage, config *models.ContestConfig,
) {
	permissions.AddPermission(perms.ObserveContestRole)
	if stage == ContestFinished {
		permissions.AddPermission(
			perms.ObserveContestProblemsRole,
			perms.ObserveContestProblemRole,
			perms.ObserveContestSolutionsRole,
			perms.SubmitContestSolutionRole,
			perms.ObserveSolutionReportTestNumber,
			perms.ObserveContestMessagesRole,
		)
		if config.StandingsKind != models.DisabledStandings {
			permissions.AddPermission(perms.ObserveContestStandingsRole)
		}
	}
}

func addContestObserverPermissions(
	permissions perms.PermissionSet, stage ContestStage, config *models.ContestConfig,
) {
	permissions.AddPermission(perms.ObserveContestRole)
	switch stage {
	case ContestStarted, ContestFinished:
		if config.StandingsKind != models.DisabledStandings {
			permissions.AddPermission(perms.ObserveContestStandingsRole)
		}
		if config.StandingsKind != models.DisabledStandings {
			permissions.AddPermission(perms.ObserveContestStandingsRole)
		}
	}
}

func getParticipantPermissions(
	config *models.ContestConfig,
	participant *models.ContestParticipant,
	now int64,
) perms.PermissionSet {
	stage := getParticipantContestTime(config, participant, now).Stage()
	permissions := perms.PermissionSet{}
	switch participant.Kind {
	case models.RegularParticipant:
		addContestRegularPermissions(permissions, stage, config)
	case models.UpsolvingParticipant:
		addContestUpsolvingPermissions(permissions, stage, config)
	case models.ManagerParticipant:
		addContestManagerPermissions(permissions)
	case models.ObserverParticipant:
		addContestObserverPermissions(permissions, stage, config)
	case models.VirtualParticipant:
		addContestVirtualPermissions(permissions, stage, config)
	}
	return permissions
}

func checkEffectiveParticipant(
	config *models.ContestConfig,
	participant *models.ContestParticipant,
	now int64,
) bool {
	stage := getParticipantContestTime(config, participant, now).Stage()
	switch participant.Kind {
	case models.RegularParticipant:
		return stage == ContestStarted
	case models.UpsolvingParticipant:
		return stage == ContestFinished
	case models.ManagerParticipant:
		return true
	case models.VirtualParticipant:
		return stage == ContestNotStarted || stage == ContestStarted
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
		ContestConfig:  config,
		Permissions:    ctx.Permissions.Clone(),
		Now:            models.GetNow(ctx),
	}
	now := c.Now.Unix()
	stage := getParticipantContestTime(&c.ContestConfig, nil, now).Stage()
	if account := ctx.Account; account != nil {
		if contest.OwnerID != 0 && account.ID == int64(contest.OwnerID) {
			c.Permissions.AddPermission(perms.DeleteContestRole)
		}
		participantRows, err := m.participants.FindByContestAccount(ctx, contest.ID, account.ID)
		if err != nil {
			return nil, fmt.Errorf("unable to build contest context: %w", err)
		}
		participants, err := db.CollectRows(participantRows)
		if err != nil {
			return nil, fmt.Errorf("unable to build contest context: %w", err)
		}
		hasRegular := false
		hasUpsolving := false
		hasManager := false
		hasVirtual := false
		for _, participant := range participants {
			for permission := range getParticipantPermissions(&config, &participant, now) {
				c.Permissions.AddPermission(permission)
			}
			switch participant.Kind {
			case models.RegularParticipant:
				hasRegular = true
			case models.UpsolvingParticipant:
				hasUpsolving = true
			case models.ManagerParticipant:
				hasManager = true
			case models.VirtualParticipant:
				hasVirtual = true
			}
		}
		c.Participants = participants
		for _, group := range ctx.GroupAccounts {
			groupParticipantRows, err := m.participants.FindByContestAccount(ctx, contest.ID, group.ID)
			if err != nil {
				return nil, fmt.Errorf("unable to build contest context: %w", err)
			}
			groupParticipants, err := db.CollectRows(groupParticipantRows)
			if err != nil {
				return nil, fmt.Errorf("unable to build contest context: %w", err)
			}
			for _, groupParticipant := range groupParticipants {
				for permission := range getParticipantPermissions(&config, &groupParticipant, now) {
					if permission == perms.DeregisterContestRole {
						// User cannot deregister group account.
						continue
					}
					c.Permissions.AddPermission(permission)
				}
				stage := getParticipantContestTime(&c.ContestConfig, &groupParticipant, now).Stage()
				switch groupParticipant.Kind {
				case models.RegularParticipant:
					if !hasRegular && (stage == ContestNotStarted || stage == ContestStarted) {
						c.Participants = append(c.Participants, models.ContestParticipant{
							ContestID: contest.ID,
							AccountID: account.ID,
							Kind:      models.RegularParticipant,
							Config:    groupParticipant.Config.Clone(),
						})
					}
					hasRegular = true
				case models.UpsolvingParticipant:
					if !hasUpsolving && stage == ContestFinished {
						c.Participants = append(c.Participants, models.ContestParticipant{
							ContestID: contest.ID,
							AccountID: account.ID,
							Kind:      models.UpsolvingParticipant,
							Config:    groupParticipant.Config.Clone(),
						})
					}
					hasUpsolving = true
				case models.ManagerParticipant:
					if !hasManager {
						c.Participants = append(c.Participants, models.ContestParticipant{
							ContestID: contest.ID,
							AccountID: account.ID,
							Kind:      models.ManagerParticipant,
							Config:    groupParticipant.Config.Clone(),
						})
					}
					hasManager = true
				}
			}
		}
		if !hasManager && contest.OwnerID != 0 && account.ID == int64(contest.OwnerID) {
			c.Participants = append(c.Participants, models.ContestParticipant{
				ContestID: contest.ID,
				AccountID: account.ID,
				Kind:      models.ManagerParticipant,
			})
			addContestManagerPermissions(c.Permissions)
		}
		// User can possibly register on contest.
		canRegister := config.EnableRegistration && c.HasPermission(perms.RegisterContestsRole)
		if !hasRegular && stage == ContestNotStarted && canRegister {
			c.Permissions.AddPermission(perms.ObserveContestRole)
			c.Permissions.AddPermission(perms.RegisterContestRole)
		}
		// User can possibly virtual register on contest.
		canVirtual := canRegister && config.EnableVirtual
		if !hasVirtual && stage >= ContestNotStarted && canVirtual {
			c.Permissions.AddPermission(perms.ObserveContestRole)
			c.Permissions.AddPermission(perms.RegisterContestVirtualRole)
		}
		// User can possibly upsolve contest.
		canUpsolving := config.EnableUpsolving && (hasRegular || canRegister)
		if !hasUpsolving && stage == ContestFinished && canUpsolving {
			// Add virtual participant for upsolving.
			c.Participants = append(c.Participants, models.ContestParticipant{
				ContestID: contest.ID,
				AccountID: account.ID,
				Kind:      models.UpsolvingParticipant,
			})
			addContestUpsolvingPermissions(c.Permissions, stage, &config)
		}
	}
	if config.EnableObserving && len(c.Participants) == 0 {
		addContestObserverPermissions(c.Permissions, stage, &config)
	}
	disableUpsolving := false
	if setting, err := m.settings.GetByKey("contests.disable_upsolving"); err == nil {
		disableUpsolving = setting.Value == "t" || setting.Value == "1" || setting.Value == "true"
	}
	c.effectivePos = len(c.Participants)
	for i := 0; i < len(c.Participants); i++ {
		if disableUpsolving && c.Participants[i].Kind == models.UpsolvingParticipant {
			continue
		}
		if checkEffectiveParticipant(&config, &c.Participants[i], now) {
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
	Contest       models.Contest
	ContestConfig models.ContestConfig
	Participants  []models.ContestParticipant
	Permissions   perms.PermissionSet
	Now           time.Time
	effectivePos  int
}

func (c *ContestContext) HasPermission(name string) bool {
	return c.Permissions.HasPermission(name)
}

func (c *ContestContext) GetEffectiveParticipant() *models.ContestParticipant {
	if c.effectivePos >= len(c.Participants) {
		return nil
	}
	return &c.Participants[c.effectivePos]
}

func (c *ContestContext) SetEffectiveParticipant(id int64) {
	now := c.Now.Unix()
	for i := range c.Participants {
		if c.Participants[i].ID == id {
			if checkEffectiveParticipant(&c.ContestConfig, &c.Participants[i], now) {
				c.effectivePos = i
			}
			break
		}
	}
}

func (c *ContestContext) GetEffectivePermissions() perms.PermissionSet {
	participant := c.GetEffectiveParticipant()
	if participant == nil {
		return perms.PermissionSet{}
	}
	return getParticipantPermissions(&c.ContestConfig, participant, c.Now.Unix())
}

func (c *ContestContext) HasEffectivePermission(name string) bool {
	return c.GetEffectivePermissions().HasPermission(name)
}

type ContestTime int64

const (
	ContestTimeNotPlanned ContestTime = -1
	ContestTimeNotStarted ContestTime = -2
	ContestTimeFinished   ContestTime = -3
)

func (c ContestTime) Before(time int64) bool {
	switch c {
	case ContestTimeNotPlanned, ContestTimeNotStarted:
		// Any time point inside contest is after not started contest.
		return true
	case ContestTimeFinished:
		// Any time poiont inside contest is before finished contest.
		return false
	default:
		if c < 0 {
			panic(fmt.Errorf("invalid contest time value: %v", c))
		}
		// Check time point for running contest.
		return int64(c) < time
	}
}

func (c ContestTime) Stage() ContestStage {
	switch c {
	case ContestTimeNotPlanned:
		return ContestNotPlanned
	case ContestTimeNotStarted:
		return ContestNotStarted
	case ContestTimeFinished:
		return ContestFinished
	default:
		if c < 0 {
			panic(fmt.Errorf("invalid contest time value: %v", c))
		}
		return ContestStarted
	}
}

func (c *ContestContext) GetEffectiveBeginTime() int64 {
	return getParticipantBeginTime(
		&c.ContestConfig,
		c.GetEffectiveParticipant(),
	)
}

func (c *ContestContext) GetEffectiveContestTime() ContestTime {
	return getParticipantContestTime(
		&c.ContestConfig,
		c.GetEffectiveParticipant(),
		c.Now.Unix(),
	)
}

// participant can be nil.
func getParticipantContestTime(
	config *models.ContestConfig,
	participant *models.ContestParticipant,
	now int64,
) ContestTime {
	beginTime := getParticipantBeginTime(config, participant)
	if beginTime == 0 {
		return ContestTimeNotPlanned
	}
	if now >= beginTime+int64(config.Duration) {
		return ContestTimeFinished
	}
	if now >= beginTime {
		// Contest started.
		return ContestTime(now - beginTime)
	}
	return ContestTimeNotStarted
}

// participant can be nil.
func getParticipantBeginTime(
	config *models.ContestConfig,
	participant *models.ContestParticipant,
) int64 {
	beginTime := int64(config.BeginTime)
	if participant == nil {
		return beginTime
	}
	switch participant.Kind {
	case models.RegularParticipant:
		var participantConfig models.RegularParticipantConfig
		if err := participant.ScanConfig(&participantConfig); err != nil {
			return beginTime
		}
		if participantConfig.BeginTime != 0 {
			beginTime = int64(participantConfig.BeginTime)
		}
	case models.VirtualParticipant:
		var participantConfig models.VirtualParticipantConfig
		if err := participant.ScanConfig(&participantConfig); err != nil {
			return beginTime
		}
		return participantConfig.BeginTime
	}
	return beginTime
}

var (
	_ context.Context   = (*ContestContext)(nil)
	_ perms.Permissions = (*ContestContext)(nil)
)

package core

import (
	"reflect"
	"time"

	"github.com/udovin/solve/models"
)

// SetupAllStores prepares all stores.
func (c *Core) SetupAllStores() {
	c.Settings = models.NewSettingStore(
		c.DB, "solve_setting", "solve_setting_event",
	)
	c.Tasks = models.NewTaskStore(
		c.DB, "solve_task", "solve_task_event",
	)
	c.Files = models.NewFileStore(
		c.DB, "solve_file", "solve_file_event",
	)
	c.Roles = models.NewRoleStore(
		c.DB, "solve_role", "solve_role_event",
	)
	c.RoleEdges = models.NewRoleEdgeStore(
		c.DB, "solve_role_edge", "solve_role_edge_event",
	)
	c.Accounts = models.NewAccountStore(
		c.DB, "solve_account", "solve_account_event",
	)
	c.AccountRoles = models.NewAccountRoleStore(
		c.DB, "solve_account_role", "solve_account_role_event",
	)
	c.Sessions = models.NewSessionStore(
		c.DB, "solve_session", "solve_session_event",
	)
	if c.Config.Security != nil {
		c.Users = models.NewUserStore(
			c.DB, "solve_user", "solve_user_event",
			c.Config.Security.PasswordSalt,
		)
	}
	c.Contests = models.NewContestStore(
		c.DB, "solve_contest", "solve_contest_event",
	)
	c.Problems = models.NewProblemStore(
		c.DB, "solve_problem", "solve_problem_event",
	)
	c.Solutions = models.NewSolutionStore(
		c.DB, "solve_solution", "solve_solution_event",
	)
	c.ContestProblems = models.NewContestProblemStore(
		c.DB, "solve_contest_problem", "solve_contest_problem_event",
	)
	c.ContestParticipants = models.NewContestParticipantStore(
		c.DB, "solve_contest_participant", "solve_contest_participant_event",
	)
	c.ContestSolutions = models.NewContestSolutionStore(
		c.DB, "solve_contest_solution", "solve_contest_solution_event",
	)
	c.Compilers = models.NewCompilerStore(
		c.DB, "solve_compiler", "solve_compiler_event",
	)
	c.Visits = models.NewVisitStore(c.DB, "solve_visit")
}

func (c *Core) startStores(start func(models.Store, time.Duration)) {
	start(c.Settings, time.Second*5)
	start(c.Tasks, time.Second)
	start(c.Files, time.Second)
	start(c.Roles, time.Second*5)
	start(c.RoleEdges, time.Second*5)
	start(c.Accounts, time.Second)
	start(c.AccountRoles, time.Second)
	start(c.Sessions, time.Second)
	start(c.Users, time.Second)
	start(c.Contests, time.Second)
	start(c.Problems, time.Second)
	start(c.Solutions, time.Second)
	start(c.ContestProblems, time.Second)
	start(c.ContestParticipants, time.Second)
	start(c.ContestSolutions, time.Second)
	start(c.Compilers, time.Second*5)
}

func (c *Core) startStoreLoops() error {
	errs := make(chan error)
	count := 0
	c.startStores(func(s models.Store, d time.Duration) {
		v := reflect.ValueOf(s)
		if s == nil || (v.Kind() == reflect.Ptr && v.IsNil()) {
			return
		}
		count++
		c.waiter.Add(1)
		go c.runStoreLoop(s, d, errs)
	})
	var err error
	for i := 0; i < count; i++ {
		lastErr := <-errs
		if lastErr != nil {
			err = lastErr
		}
	}
	return err
}

func (c *Core) runStoreLoop(
	s models.Store, d time.Duration, errs chan<- error,
) {
	defer c.waiter.Done()
	err := s.Init(c.context)
	errs <- err
	if err != nil {
		c.Logger().Error("Cannot init store", err)
		return
	}
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-c.context.Done():
			return
		case <-ticker.C:
			if err := s.Sync(c.context); err != nil {
				c.Logger().Warn("Cannot sync store", err)
			}
		}
	}
}

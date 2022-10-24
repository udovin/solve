package core

import (
	"context"
	"reflect"
	"sync"
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
	c.ProblemParts = models.NewProblemPartStore(
		c.DB, "solve_problem_part", "solve_problem_part_event",
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

func (c *Core) startStores(start func(models.Store, string, time.Duration)) {
	start(c.Settings, "settings", time.Second*5)
	start(c.Tasks, "tasks", time.Second)
	start(c.Files, "files", time.Second)
	start(c.Roles, "roles", time.Second*5)
	start(c.RoleEdges, "role_edges", time.Second*5)
	start(c.Accounts, "accounts", time.Second)
	start(c.AccountRoles, "account_roles", time.Second)
	start(c.Sessions, "sessions", time.Second)
	start(c.Users, "users", time.Second)
	start(c.Contests, "contests", time.Second)
	start(c.Problems, "problems", time.Second)
	start(c.ProblemParts, "problem_parts", time.Second)
	start(c.Solutions, "solutions", time.Second)
	start(c.ContestProblems, "contest_problems", time.Second)
	start(c.ContestParticipants, "contest_participants", time.Second)
	start(c.ContestSolutions, "contest_solutions", time.Second)
	start(c.Compilers, "compilers", time.Second*5)
}

func (c *Core) startStoreLoops() (err error) {
	once := sync.Once{}
	var waiter sync.WaitGroup
	defer waiter.Wait()
	start := func(store models.Store, name string, delay time.Duration) {
		if isNil(store) {
			return
		}
		waiter.Add(1)
		c.startCoreTask(func() {
			defer waiter.Done()
			c.Logger().Debug("Store init started", Any("store", name))
			if errStore := store.Init(c.context); errStore != nil {
				if errStore != context.Canceled {
					c.Logger().Error("Store init failed", errStore, Any("store", name))
				} else {
					c.Logger().Warn("Store init canceled", Any("store", name))
				}
				once.Do(func() { err = errStore })
				// Abort core.
				c.cancel()
				return
			}
			c.Logger().Debug("Store init finished", Any("store", name))
			c.startCoreTask(func() {
				c.storeLoop(store, name, delay)
			})
		})
	}
	c.startStores(start)
	return
}

func (c *Core) storeLoop(store models.Store, name string, delay time.Duration) {
	c.Logger().Debug("Store sync loop started", Any("store", name))
	defer c.Logger().Debug("Store sync loop stopped", Any("store", name))
	updateTime := time.Now()
	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for {
		select {
		case <-c.context.Done():
			return
		case <-ticker.C:
			if err := store.Sync(c.context); err != nil {
				if time.Since(updateTime) > delay*15 {
					c.Logger().Error("Cannot sync store", err, Any("store", name))
					// Abort core.
					c.cancel()
					return
				}
				c.Logger().Warn("Cannot sync store", err, Any("store", name))
			} else {
				updateTime = time.Now()
			}
		}
	}
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Ptr && rv.IsNil()
}

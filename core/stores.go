package core

import (
	"log"
	"reflect"
	"time"

	"github.com/udovin/solve/db"
	"github.com/udovin/solve/models"
)

// SetupAllStores prepares all stores.
func (c *Core) SetupAllStores() error {
	salt, err := c.Config.Security.PasswordSalt.Secret()
	if err != nil {
		return err
	}
	dialect := db.SQLite
	c.Tasks = models.NewTaskStore(
		"solve_task", "solve_task_event", dialect,
	)
	c.Roles = models.NewRoleStore(
		"solve_role", "solve_role_event", dialect,
	)
	c.RoleEdges = models.NewRoleEdgeStore(
		"solve_role_edge", "solve_role_edge_event", dialect,
	)
	c.Accounts = models.NewAccountStore(
		"solve_account", "solve_account_event", dialect,
	)
	c.AccountRoles = models.NewAccountRoleStore(
		"solve_account_role", "solve_account_role_event", dialect,
	)
	c.Sessions = models.NewSessionStore(
		"solve_session", "solve_session_event", dialect,
	)
	c.Users = models.NewUserStore(
		"solve_user", "solve_user_event", salt, dialect,
	)
	c.Contests = models.NewContestStore(
		"solve_contest", "solve_contest_event", dialect,
	)
	c.Problems = models.NewProblemStore(
		"solve_problem", "solve_problem_event", dialect,
	)
	c.Solutions = models.NewSolutionStore(
		"solve_solution", "solve_solution_event", dialect,
	)
	c.ContestProblems = models.NewContestProblemStore(
		"solve_contest_problem", "solve_contest_problem_event", dialect,
	)
	c.Visits = models.NewVisitStore("solve_visit", dialect)
	return nil
}

func (c *Core) startStores(start func(models.Store, time.Duration)) {
	start(c.Tasks, time.Second)
	start(c.Roles, time.Second)
	start(c.RoleEdges, time.Second)
	start(c.Accounts, time.Second)
	start(c.AccountRoles, time.Second)
	start(c.Sessions, time.Second)
	start(c.Users, time.Second)
	start(c.Contests, time.Second)
	start(c.Problems, time.Second)
	start(c.Solutions, time.Second)
	start(c.ContestProblems, time.Second)
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
			log.Println("Error:", lastErr)
			err = lastErr
		}
	}
	return err
}

func (c *Core) runStoreLoop(
	s models.Store, d time.Duration, errs chan<- error,
) {
	defer c.waiter.Done()
	err := c.WithTx(c.context, s.InitTx)
	errs <- err
	if err != nil {
		return
	}
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-c.context.Done():
			return
		case <-ticker.C:
			if err := c.WithTx(c.context, s.SyncTx); err != nil {
				log.Println("Error:", err)
			}
		}
	}
}

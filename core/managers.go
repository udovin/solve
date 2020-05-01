package core

import (
	"log"
	"reflect"
	"time"

	"github.com/udovin/solve/models"
)

// SetupInvokerManagers prepares managers for running invoker.
func (c *Core) SetupInvokerManagers() {
	dialect := c.Dialect()
	c.Actions = models.NewActionManager(
		"solve_action", "solve_action_event", dialect,
	)
}

// SetupAllManagers prepares all managers.
func (c *Core) SetupAllManagers() error {
	salt, err := c.Config.Security.PasswordSalt.Secret()
	if err != nil {
		return err
	}
	dialect := c.Dialect()
	c.Actions = models.NewActionManager(
		"solve_action", "solve_action_event", dialect,
	)
	c.Roles = models.NewRoleManager(
		"solve_role", "solve_role_event", dialect,
	)
	c.RoleEdges = models.NewRoleEdgeManager(
		"solve_role_edge", "solve_role_edge_event", dialect,
	)
	c.Accounts = models.NewAccountManager(
		"solve_account", "solve_account_event", dialect,
	)
	c.AccountRoles = models.NewAccountRoleManager(
		"solve_account_role", "solve_account_role_event", dialect,
	)
	c.Sessions = models.NewSessionManager(
		"solve_session", "solve_session_event", dialect,
	)
	c.Users = models.NewUserManager(
		"solve_user", "solve_user_event", salt, dialect,
	)
	c.UserFields = models.NewUserFieldManager(
		"solve_user_field", "solve_user_field_event", dialect,
	)
	c.Contests = models.NewContestManager(
		"solve_contest", "solve_contest_event", dialect,
	)
	c.Problems = models.NewProblemManager(
		"solve_problem", "solve_problem_event", dialect,
	)
	c.ContestProblems = models.NewContestProblemManager(
		"solve_contest_problem", "solve_contest_problem_event", dialect,
	)
	c.Visits = models.NewVisitManager("solve_visit", dialect)
	return nil
}

func (c *Core) startManagers(start func(models.Manager, time.Duration)) {
	start(c.Actions, time.Second)
	start(c.Roles, time.Second)
	start(c.RoleEdges, time.Second)
	start(c.Accounts, time.Second)
	start(c.AccountRoles, time.Second)
	start(c.Sessions, time.Second)
	start(c.Users, time.Second)
	start(c.UserFields, time.Second)
	start(c.Contests, time.Second)
	start(c.Problems, time.Second)
	start(c.ContestProblems, time.Second)
}

func (c *Core) startManagerLoops() error {
	errs := make(chan error)
	count := 0
	c.startManagers(func(m models.Manager, d time.Duration) {
		v := reflect.ValueOf(m)
		if m == nil || (v.Kind() == reflect.Ptr && v.IsNil()) {
			return
		}
		count++
		c.waiter.Add(1)
		go c.runManagerLoop(m, d, errs)
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

func (c *Core) runManagerLoop(
	m models.Manager, d time.Duration, errs chan<- error,
) {
	defer c.waiter.Done()
	err := c.WithTx(c.context, m.InitTx)
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
			if err := c.WithTx(c.context, m.SyncTx); err != nil {
				log.Println("Error:", err)
			}
		}
	}
}

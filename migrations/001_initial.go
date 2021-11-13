package migrations

import (
	"database/sql"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/db/schema"
	"github.com/udovin/solve/models"
)

type m001 struct{}

func (m *m001) Name() string {
	return "001_initial"
}

var m001Tables = []schema.Table{
	{
		Name: "solve_task",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true},
			{Name: "status", Type: schema.Int64},
			{Name: "kind", Type: schema.Int64},
			{Name: "config", Type: schema.JSON},
			{Name: "state", Type: schema.JSON},
			{Name: "expire_time", Type: schema.Int64},
		},
	},
	{
		Name: "solve_task_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true},
			{Name: "event_type", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "id", Type: schema.Int64},
			{Name: "status", Type: schema.Int64},
			{Name: "kind", Type: schema.Int64},
			{Name: "config", Type: schema.JSON},
			{Name: "state", Type: schema.JSON},
			{Name: "expire_time", Type: schema.Int64},
		},
	},
	{
		Name: "solve_role",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true},
			{Name: "code", Type: schema.String},
		},
	},
	{
		Name: "solve_role_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true},
			{Name: "event_type", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "id", Type: schema.Int64},
			{Name: "code", Type: schema.String},
		},
	},
	{
		Name: "solve_role_edge",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true},
			{Name: "role_id", Type: schema.Int64},
			{Name: "child_id", Type: schema.Int64},
		},
	},
	{
		Name: "solve_role_edge_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true},
			{Name: "event_type", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "id", Type: schema.Int64},
			{Name: "role_id", Type: schema.Int64},
			{Name: "child_id", Type: schema.Int64},
		},
	},
	{
		Name: "solve_account",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true},
			{Name: "kind", Type: schema.Int64},
		},
	},
	{
		Name: "solve_account_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true},
			{Name: "event_type", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "id", Type: schema.Int64},
			{Name: "kind", Type: schema.Int64},
		},
	},
	{
		Name: "solve_account_role",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true},
			{Name: "account_id", Type: schema.Int64},
			{Name: "role_id", Type: schema.Int64},
		},
	},
	{
		Name: "solve_account_role_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true},
			{Name: "event_type", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "id", Type: schema.Int64},
			{Name: "account_id", Type: schema.Int64},
			{Name: "role_id", Type: schema.Int64},
		},
	},
	{
		Name: "solve_session",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true},
			{Name: "account_id", Type: schema.Int64},
			{Name: "secret", Type: schema.String},
			{Name: "create_time", Type: schema.Int64},
			{Name: "expire_time", Type: schema.Int64},
		},
	},
	{
		Name: "solve_session_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true},
			{Name: "event_type", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "id", Type: schema.Int64},
			{Name: "account_id", Type: schema.Int64},
			{Name: "secret", Type: schema.String},
			{Name: "create_time", Type: schema.Int64},
			{Name: "expire_time", Type: schema.Int64},
		},
	},
	{
		Name: "solve_user",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true},
			{Name: "account_id", Type: schema.Int64},
			{Name: "login", Type: schema.String},
			{Name: "password_hash", Type: schema.String},
			{Name: "password_salt", Type: schema.String},
			{Name: "email", Type: schema.String, Nullable: true},
			{Name: "first_name", Type: schema.String, Nullable: true},
			{Name: "last_name", Type: schema.String, Nullable: true},
			{Name: "middle_name", Type: schema.String, Nullable: true},
		},
	},
	{
		Name: "solve_user_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true},
			{Name: "event_type", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "id", Type: schema.Int64},
			{Name: "account_id", Type: schema.Int64},
			{Name: "login", Type: schema.String},
			{Name: "password_hash", Type: schema.String},
			{Name: "password_salt", Type: schema.String},
			{Name: "email", Type: schema.String, Nullable: true},
			{Name: "first_name", Type: schema.String, Nullable: true},
			{Name: "last_name", Type: schema.String, Nullable: true},
			{Name: "middle_name", Type: schema.String, Nullable: true},
		},
	},
	{
		Name: "solve_contest",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true},
			{Name: "owner_id", Type: schema.Int64, Nullable: true},
			{Name: "config", Type: schema.JSON},
			{Name: "title", Type: schema.String},
		},
	},
	{
		Name: "solve_contest_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true},
			{Name: "event_type", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "id", Type: schema.Int64},
			{Name: "owner_id", Type: schema.Int64, Nullable: true},
			{Name: "config", Type: schema.JSON},
			{Name: "title", Type: schema.String},
		},
	},
	{
		Name: "solve_problem",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true},
			{Name: "owner_id", Type: schema.Int64, Nullable: true},
			{Name: "config", Type: schema.JSON},
			{Name: "title", Type: schema.String},
		},
	},
	{
		Name: "solve_problem_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true},
			{Name: "event_type", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "id", Type: schema.Int64},
			{Name: "owner_id", Type: schema.Int64, Nullable: true},
			{Name: "config", Type: schema.JSON},
			{Name: "title", Type: schema.String},
		},
	},
	{
		Name: "solve_solution",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true},
			{Name: "problem_id", Type: schema.Int64},
			{Name: "author_id", Type: schema.Int64},
		},
	},
	{
		Name: "solve_solution_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true},
			{Name: "event_type", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "id", Type: schema.Int64},
			{Name: "problem_id", Type: schema.Int64},
			{Name: "author_id", Type: schema.Int64},
		},
	},
	{
		Name: "solve_contest_problem",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true},
			{Name: "contest_id", Type: schema.Int64},
			{Name: "problem_id", Type: schema.Int64},
			{Name: "code", Type: schema.String},
		},
	},
	{
		Name: "solve_contest_problem_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true},
			{Name: "event_type", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "id", Type: schema.Int64},
			{Name: "contest_id", Type: schema.Int64},
			{Name: "problem_id", Type: schema.Int64},
			{Name: "code", Type: schema.String},
		},
	},
	{
		Name: "solve_visit",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true},
			{Name: "time", Type: schema.Int64},
			{Name: "account_id", Type: schema.Int64, Nullable: true},
			{Name: "session_id", Type: schema.Int64, Nullable: true},
			{Name: "host", Type: schema.String},
			{Name: "protocol", Type: schema.String},
			{Name: "method", Type: schema.String},
			{Name: "remote_addr", Type: schema.String},
			{Name: "user_agent", Type: schema.String},
			{Name: "path", Type: schema.String},
			{Name: "real_ip", Type: schema.String},
			{Name: "status", Type: schema.Int64},
		},
	},
}

func (m *m001) Apply(c *core.Core, tx *sql.Tx) error {
	for _, table := range m001Tables {
		query, err := table.BuildCreateSQL(c.DB.Dialect())
		if err != nil {
			return err
		}
		if _, err := tx.Exec(query); err != nil {
			return err
		}
	}
	return m.createRoles(c, tx)
}

func (m *m001) Unapply(c *core.Core, tx *sql.Tx) error {
	for i := 0; i < len(m001Tables); i++ {
		table := m001Tables[len(m001Tables)-i-1]
		query, err := table.BuildDropSQL(c.DB.Dialect(), false)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (m *m001) createRoles(c *core.Core, tx *sql.Tx) error {
	roles := map[string]int64{}
	create := func(code string) error {
		role, err := c.Roles.CreateTx(tx, models.Role{Code: code})
		if err == nil {
			roles[role.Code] = role.ID
		}
		return err
	}
	join := func(child, parent string) error {
		_, err := c.RoleEdges.CreateTx(tx, models.RoleEdge{
			RoleID:  roles[parent],
			ChildID: roles[child],
		})
		return err
	}
	allRoles := []string{
		models.LoginRole,
		models.LogoutRole,
		models.RegisterRole,
		models.StatusRole,
		models.ObserveRolesRole,
		models.CreateRoleRole,
		models.DeleteRoleRole,
		models.ObserveRoleRolesRole,
		models.CreateRoleRoleRole,
		models.DeleteRoleRoleRole,
		models.ObserveUserRolesRole,
		models.CreateUserRoleRole,
		models.DeleteUserRoleRole,
		models.ObserveUserRole,
		models.UpdateUserRole,
		models.ObserveUserEmailRole,
		models.ObserveUserFirstNameRole,
		models.ObserveUserLastNameRole,
		models.ObserveUserMiddleNameRole,
		models.ObserveUserSessionsRole,
		models.UpdateUserPasswordRole,
		models.UpdateUserEmailRole,
		models.UpdateUserFirstNameRole,
		models.UpdateUserLastNameRole,
		models.UpdateUserMiddleNameRole,
		models.ObserveSessionRole,
		models.DeleteSessionRole,
		models.ObserveProblemRole,
		models.CreateProblemRole,
		models.UpdateProblemRole,
		models.DeleteProblemRole,
		models.ObserveProblemsRole,
		models.ObserveContestRole,
		models.ObserveContestProblemsRole,
		models.ObserveContestProblemRole,
		models.CreateContestRole,
		models.UpdateContestRole,
		models.DeleteContestRole,
		models.ObserveContestsRole,
	}
	allGroups := []string{
		models.GuestGroupRole,
		models.UserGroupRole,
		"admin_group",
	}
	for _, role := range allRoles {
		if err := create(role); err != nil {
			return err
		}
	}
	for _, role := range allGroups {
		if err := create(role); err != nil {
			return err
		}
	}
	for _, role := range []string{
		models.LoginRole,
		models.RegisterRole,
		models.StatusRole,
		models.ObserveUserRole,
		models.ObserveProblemsRole,
		models.ObserveContestsRole,
	} {
		if err := join(role, models.GuestGroupRole); err != nil {
			return err
		}
	}
	for _, role := range []string{
		models.LoginRole,
		models.LogoutRole,
		models.StatusRole,
		models.ObserveUserRole,
		models.ObserveProblemsRole,
		models.ObserveContestsRole,
	} {
		if err := join(role, models.UserGroupRole); err != nil {
			return err
		}
	}
	for _, role := range allRoles {
		if err := join(role, "admin_group"); err != nil {
			return err
		}
	}
	return nil
}

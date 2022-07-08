package migrations

import (
	"context"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/db/schema"
	"github.com/udovin/solve/models"
)

type m001 struct{}

func (m *m001) Name() string {
	return "001_initial"
}

var m001Tables = []schema.Table{
	{
		Name: "solve_setting",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "key", Type: schema.String},
			{Name: "value", Type: schema.String},
		},
	},
	{
		Name: "solve_setting_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "key", Type: schema.String},
			{Name: "value", Type: schema.String},
		},
	},
	{
		Name: "solve_task",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
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
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
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
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: schema.String},
		},
	},
	{
		Name: "solve_role_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "name", Type: schema.String},
		},
	},
	{
		Name: "solve_role_edge",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "role_id", Type: schema.Int64},
			{Name: "child_id", Type: schema.Int64},
		},
	},
	{
		Name: "solve_role_edge_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "role_id", Type: schema.Int64},
			{Name: "child_id", Type: schema.Int64},
		},
	},
	{
		Name: "solve_account",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "kind", Type: schema.Int64},
		},
	},
	{
		Name: "solve_account_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "kind", Type: schema.Int64},
		},
	},
	{
		Name: "solve_account_role",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "account_id", Type: schema.Int64},
			{Name: "role_id", Type: schema.Int64},
		},
	},
	{
		Name: "solve_account_role_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "account_id", Type: schema.Int64},
			{Name: "role_id", Type: schema.Int64},
		},
	},
	{
		Name: "solve_session",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "account_id", Type: schema.Int64},
			{Name: "secret", Type: schema.String},
			{Name: "create_time", Type: schema.Int64},
			{Name: "expire_time", Type: schema.Int64},
			{Name: "remote_addr", Type: schema.String},
			{Name: "user_agent", Type: schema.String},
		},
	},
	{
		Name: "solve_session_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "account_id", Type: schema.Int64},
			{Name: "secret", Type: schema.String},
			{Name: "create_time", Type: schema.Int64},
			{Name: "expire_time", Type: schema.Int64},
			{Name: "remote_addr", Type: schema.String},
			{Name: "user_agent", Type: schema.String},
		},
	},
	{
		Name: "solve_user",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
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
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
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
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "owner_id", Type: schema.Int64, Nullable: true},
			{Name: "config", Type: schema.JSON},
			{Name: "title", Type: schema.String},
		},
	},
	{
		Name: "solve_contest_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "owner_id", Type: schema.Int64, Nullable: true},
			{Name: "config", Type: schema.JSON},
			{Name: "title", Type: schema.String},
		},
	},
	{
		Name: "solve_problem",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "owner_id", Type: schema.Int64, Nullable: true},
			{Name: "config", Type: schema.JSON},
			{Name: "title", Type: schema.String},
		},
	},
	{
		Name: "solve_problem_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "owner_id", Type: schema.Int64, Nullable: true},
			{Name: "config", Type: schema.JSON},
			{Name: "title", Type: schema.String},
		},
	},
	{
		Name: "solve_solution",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "problem_id", Type: schema.Int64},
			{Name: "compiler_id", Type: schema.Int64},
			{Name: "author_id", Type: schema.Int64},
			{Name: "report", Type: schema.JSON},
			{Name: "create_time", Type: schema.Int64},
		},
	},
	{
		Name: "solve_solution_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "problem_id", Type: schema.Int64},
			{Name: "compiler_id", Type: schema.Int64},
			{Name: "author_id", Type: schema.Int64},
			{Name: "report", Type: schema.JSON},
			{Name: "create_time", Type: schema.Int64},
		},
	},
	{
		Name: "solve_contest_problem",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "contest_id", Type: schema.Int64},
			{Name: "problem_id", Type: schema.Int64},
			{Name: "code", Type: schema.String},
		},
	},
	{
		Name: "solve_contest_problem_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "contest_id", Type: schema.Int64},
			{Name: "problem_id", Type: schema.Int64},
			{Name: "code", Type: schema.String},
		},
	},
	{
		Name: "solve_contest_participant",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "contest_id", Type: schema.Int64},
			{Name: "account_id", Type: schema.Int64},
			{Name: "kind", Type: schema.Int64},
			{Name: "config", Type: schema.JSON},
		},
	},
	{
		Name: "solve_contest_participant_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "contest_id", Type: schema.Int64},
			{Name: "account_id", Type: schema.Int64},
			{Name: "kind", Type: schema.Int64},
			{Name: "config", Type: schema.JSON},
		},
	},
	{
		Name: "solve_contest_solution",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "solution_id", Type: schema.Int64},
			{Name: "contest_id", Type: schema.Int64},
			{Name: "participant_id", Type: schema.Int64},
			{Name: "problem_id", Type: schema.Int64},
		},
	},
	{
		Name: "solve_contest_solution_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "solution_id", Type: schema.Int64},
			{Name: "contest_id", Type: schema.Int64},
			{Name: "participant_id", Type: schema.Int64},
			{Name: "problem_id", Type: schema.Int64},
		},
	},
	{
		Name: "solve_compiler",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "owner_id", Type: schema.Int64, Nullable: true},
			{Name: "name", Type: schema.String},
			{Name: "config", Type: schema.JSON},
		},
	},
	{
		Name: "solve_compiler_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "owner_id", Type: schema.Int64, Nullable: true},
			{Name: "name", Type: schema.String},
			{Name: "config", Type: schema.JSON},
		},
	},
	{
		Name: "solve_visit",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
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

func (m *m001) Apply(ctx context.Context, c *core.Core) error {
	tx := db.GetRunner(ctx, c.DB)
	for _, table := range m001Tables {
		query, err := table.BuildCreateSQL(c.DB.Dialect(), false)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	return m.createRoles(ctx, c)
}

func (m *m001) Unapply(ctx context.Context, c *core.Core) error {
	tx := db.GetRunner(ctx, c.DB)
	for i := 0; i < len(m001Tables); i++ {
		table := m001Tables[len(m001Tables)-i-1]
		query, err := table.BuildDropSQL(c.DB.Dialect(), false)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func (m *m001) createRoles(ctx context.Context, c *core.Core) error {
	roles := map[string]int64{}
	create := func(name string) error {
		role := models.Role{Name: name}
		err := c.Roles.Create(ctx, &role)
		if err == nil {
			roles[role.Name] = role.ID
		}
		return err
	}
	join := func(child, parent string) error {
		edge := models.RoleEdge{
			RoleID:  roles[parent],
			ChildID: roles[child],
		}
		return c.RoleEdges.Create(ctx, &edge)
	}
	allRoles := models.GetBuiltInRoles()
	allGroups := []string{
		"guest_group",
		"user_group",
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
		models.ObserveSolutionsRole,
	} {
		if err := join(role, "guest_group"); err != nil {
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
		models.ObserveSolutionsRole,
	} {
		if err := join(role, "user_group"); err != nil {
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

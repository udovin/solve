package migrations

import (
	"github.com/udovin/solve/db"
	"github.com/udovin/solve/db/schema"
)

func init() {
	db.RegisterMigration(db.NewMigration("001_initial", m001))
}

var m001 = []schema.Operation{
	schema.CreateTable{
		Name: "solve_setting",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "key", Type: schema.String},
			{Name: "value", Type: schema.String},
		},
	},
	schema.CreateIndex{
		Table:   "solve_setting",
		Columns: []string{"key"},
		Unique:  true,
	},
	schema.CreateTable{
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
	schema.CreateTable{
		Name: "solve_task",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "status", Type: schema.Int64},
			{Name: "kind", Type: schema.Int64},
			{Name: "config", Type: schema.JSON},
			{Name: "state", Type: schema.JSON},
			{Name: "expire_time", Type: schema.Int64, Nullable: true},
		},
	},
	schema.CreateTable{
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
			{Name: "expire_time", Type: schema.Int64, Nullable: true},
		},
	},
	schema.CreateTable{
		Name: "solve_file",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "status", Type: schema.Int64},
			{Name: "expire_time", Type: schema.Int64, Nullable: true},
			{Name: "name", Type: schema.String},
			{Name: "size", Type: schema.Int64},
			{Name: "path", Type: schema.String},
		},
	},
	schema.CreateTable{
		Name: "solve_file_event",
		Columns: []schema.Column{
			{Name: "event_id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "event_kind", Type: schema.Int64},
			{Name: "event_time", Type: schema.Int64},
			{Name: "event_account_id", Type: schema.Int64, Nullable: true},
			{Name: "id", Type: schema.Int64},
			{Name: "status", Type: schema.Int64},
			{Name: "expire_time", Type: schema.Int64, Nullable: true},
			{Name: "name", Type: schema.String},
			{Name: "size", Type: schema.Int64},
			{Name: "path", Type: schema.String},
		},
	},
	schema.CreateTable{
		Name: "solve_role",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: schema.String},
		},
	},
	schema.CreateIndex{
		Table:   "solve_role",
		Columns: []string{"name"},
		Unique:  true,
	},
	schema.CreateTable{
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
	schema.CreateTable{
		Name: "solve_role_edge",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "role_id", Type: schema.Int64},
			{Name: "child_id", Type: schema.Int64},
		},
		ForeignKeys: []schema.ForeignKey{
			{Column: "role_id", ParentTable: "solve_role", ParentColumn: "id"},
			{Column: "child_id", ParentTable: "solve_role", ParentColumn: "id"},
		},
	},
	schema.CreateIndex{
		Table:   "solve_role_edge",
		Columns: []string{"role_id", "child_id"},
		Unique:  true,
	},
	schema.CreateTable{
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
	schema.CreateTable{
		Name: "solve_account",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "kind", Type: schema.Int64},
		},
	},
	schema.CreateTable{
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
	schema.CreateTable{
		Name: "solve_account_role",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "account_id", Type: schema.Int64},
			{Name: "role_id", Type: schema.Int64},
		},
		ForeignKeys: []schema.ForeignKey{
			{Column: "account_id", ParentTable: "solve_account", ParentColumn: "id"},
			{Column: "role_id", ParentTable: "solve_role", ParentColumn: "id"},
		},
	},
	schema.CreateIndex{
		Table:   "solve_account_role",
		Columns: []string{"account_id", "role_id"},
		Unique:  true,
	},
	schema.CreateTable{
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
	schema.CreateTable{
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
		ForeignKeys: []schema.ForeignKey{
			{Column: "account_id", ParentTable: "solve_account", ParentColumn: "id"},
		},
	},
	schema.CreateTable{
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
	schema.CreateTable{
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
		ForeignKeys: []schema.ForeignKey{
			{Column: "account_id", ParentTable: "solve_account", ParentColumn: "id"},
		},
	},
	schema.CreateIndex{
		Table:   "solve_user",
		Columns: []string{"account_id"},
		Unique:  true,
	},
	schema.CreateIndex{
		Table:   "solve_user",
		Columns: []string{"login"},
		Unique:  true,
	},
	schema.CreateTable{
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
	schema.CreateTable{
		Name: "solve_contest",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "owner_id", Type: schema.Int64, Nullable: true},
			{Name: "config", Type: schema.JSON},
			{Name: "title", Type: schema.String},
		},
		ForeignKeys: []schema.ForeignKey{
			{Column: "owner_id", ParentTable: "solve_account", ParentColumn: "id"},
		},
	},
	schema.CreateTable{
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
	schema.CreateTable{
		Name: "solve_compiler",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "owner_id", Type: schema.Int64, Nullable: true},
			{Name: "name", Type: schema.String},
			{Name: "config", Type: schema.JSON},
			{Name: "image_id", Type: schema.Int64},
		},
		ForeignKeys: []schema.ForeignKey{
			{Column: "owner_id", ParentTable: "solve_account", ParentColumn: "id"},
			{Column: "image_id", ParentTable: "solve_file", ParentColumn: "id"},
		},
	},
	schema.CreateIndex{
		Table:   "solve_compiler",
		Columns: []string{"name"},
		Unique:  true,
	},
	schema.CreateTable{
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
			{Name: "image_id", Type: schema.Int64},
		},
	},
	schema.CreateTable{
		Name: "solve_problem",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "owner_id", Type: schema.Int64, Nullable: true},
			{Name: "config", Type: schema.JSON},
			{Name: "title", Type: schema.String},
			{Name: "package_id", Type: schema.Int64},
		},
		ForeignKeys: []schema.ForeignKey{
			{Column: "owner_id", ParentTable: "solve_account", ParentColumn: "id"},
			{Column: "package_id", ParentTable: "solve_file", ParentColumn: "id"},
		},
	},
	schema.CreateTable{
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
			{Name: "package_id", Type: schema.Int64},
		},
	},
	schema.CreateTable{
		Name: "solve_solution",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "problem_id", Type: schema.Int64},
			{Name: "compiler_id", Type: schema.Int64},
			{Name: "author_id", Type: schema.Int64},
			{Name: "report", Type: schema.JSON},
			{Name: "create_time", Type: schema.Int64},
			{Name: "content", Type: schema.String, Nullable: true},
			{Name: "content_id", Type: schema.Int64, Nullable: true},
		},
		ForeignKeys: []schema.ForeignKey{
			{Column: "problem_id", ParentTable: "solve_problem", ParentColumn: "id"},
			{Column: "compiler_id", ParentTable: "solve_compiler", ParentColumn: "id"},
			{Column: "author_id", ParentTable: "solve_account", ParentColumn: "id"},
			{Column: "content_id", ParentTable: "solve_file", ParentColumn: "id"},
		},
	},
	schema.CreateTable{
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
			{Name: "content", Type: schema.String, Nullable: true},
			{Name: "content_id", Type: schema.Int64, Nullable: true},
		},
	},
	schema.CreateTable{
		Name: "solve_contest_problem",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "contest_id", Type: schema.Int64},
			{Name: "problem_id", Type: schema.Int64},
			{Name: "code", Type: schema.String},
		},
		ForeignKeys: []schema.ForeignKey{
			{Column: "contest_id", ParentTable: "solve_contest", ParentColumn: "id"},
			{Column: "problem_id", ParentTable: "solve_problem", ParentColumn: "id"},
		},
	},
	schema.CreateIndex{
		Table:   "solve_contest_problem",
		Columns: []string{"contest_id", "problem_id"},
		Unique:  true,
	},
	schema.CreateIndex{
		Table:   "solve_contest_problem",
		Columns: []string{"contest_id", "code"},
		Unique:  true,
	},
	schema.CreateTable{
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
	schema.CreateTable{
		Name: "solve_contest_participant",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "contest_id", Type: schema.Int64},
			{Name: "account_id", Type: schema.Int64},
			{Name: "kind", Type: schema.Int64},
			{Name: "config", Type: schema.JSON},
		},
		ForeignKeys: []schema.ForeignKey{
			{Column: "contest_id", ParentTable: "solve_contest", ParentColumn: "id"},
			{Column: "account_id", ParentTable: "solve_account", ParentColumn: "id"},
		},
	},
	schema.CreateIndex{
		Table:   "solve_contest_participant",
		Columns: []string{"contest_id", "account_id", "kind"},
		Unique:  true,
	},
	schema.CreateTable{
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
	schema.CreateTable{
		Name: "solve_contest_solution",
		Columns: []schema.Column{
			{Name: "id", Type: schema.Int64, PrimaryKey: true, AutoIncrement: true},
			{Name: "contest_id", Type: schema.Int64},
			{Name: "solution_id", Type: schema.Int64},
			{Name: "participant_id", Type: schema.Int64},
			{Name: "problem_id", Type: schema.Int64},
		},
		ForeignKeys: []schema.ForeignKey{
			{Column: "contest_id", ParentTable: "solve_contest", ParentColumn: "id"},
			{Column: "solution_id", ParentTable: "solve_solution", ParentColumn: "id"},
			{Column: "participant_id", ParentTable: "solve_contest_participant", ParentColumn: "id"},
			{Column: "problem_id", ParentTable: "solve_contest_problem", ParentColumn: "id"},
		},
	},
	schema.CreateIndex{
		Table:   "solve_contest_solution",
		Columns: []string{"solution_id"},
		Unique:  true,
	},
	schema.CreateTable{
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
	schema.CreateTable{
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

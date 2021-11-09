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
}

const m001Apply = `
-- models.User
CREATE TABLE "solve_user"
(
	"id" integer PRIMARY KEY,
	"account_id" integer NOT NULL,
	"login" VARCHAR(255) NOT NULL,
	"password_hash" VARCHAR(255) NOT NULL,
	"password_salt" TEXT NOT NULL,
	"email" VARCHAR(255),
	"first_name" VARCHAR(255),
	"last_name" VARCHAR(255),
	"middle_name" VARCHAR(255)
);
-- models.UserEvent
CREATE TABLE "solve_user_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"account_id" integer NOT NULL,
	"login" VARCHAR(255) NOT NULL,
	"password_hash" VARCHAR(255) NOT NULL,
	"password_salt" TEXT NOT NULL,
	"email" VARCHAR(255),
	"first_name" VARCHAR(255),
	"last_name" VARCHAR(255),
	"middle_name" VARCHAR(255)
);
-- models.Contest
CREATE TABLE "solve_contest"
(
	"id" integer PRIMARY KEY,
	"owner_id" integer,
	"config" blob NOT NULL,
	"title" VARCHAR(255)
);
-- models.ContestEvent
CREATE TABLE "solve_contest_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"owner_id" integer,
	"config" blob NOT NULL,
	"title" VARCHAR(255)
);
-- models.Problem
CREATE TABLE "solve_problem"
(
	"id" integer PRIMARY KEY,
	"owner_id" integer,
	"config" blob NOT NULL,
	"title" VARCHAR(255) NOT NULL
);
-- models.ProblemEvent
CREATE TABLE "solve_problem_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"owner_id" integer,
	"config" blob NOT NULL,
	"title" VARCHAR(255) NOT NULL
);
-- models.Solution
CREATE TABLE "solve_solution"
(
	"id" integer PRIMARY KEY,
	"problem_id" integer NOT NULL,
	"author_id" integer NOT NULL
);
-- models.SolutionEvent
CREATE TABLE "solve_solution_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"problem_id" integer NOT NULL,
	"author_id" integer NOT NULL
);
-- models.ContestProblem
CREATE TABLE "solve_contest_problem"
(
	"id" integer PRIMARY KEY,
	"contest_id" integer NOT NULL,
	"problem_id" integer NOT NULL,
	"code" VARCHAR(32) NOT NULL
);
-- models.ContestProblemEvent
CREATE TABLE "solve_contest_problem_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"contest_id" integer NOT NULL,
	"problem_id" integer NOT NULL,
	"code" VARCHAR(32) NOT NULL
);
-- models.Visit
CREATE TABLE "solve_visit"
(
	"id" integer PRIMARY KEY,
	"time" bigint NOT NULL,
	"account_id" integer,
	"session_id" integer,
	"host" varchar(255) NOT NULL,
	"protocol" varchar(255) NOT NULL,
	"method" varchar(255) NOT NULL,
	"remote_addr" varchar(255) NOT NULL,
	"user_agent" varchar(255) NOT NULL,
	"path" varchar(255) NOT NULL,
	"real_ip" varchar(255) NOT NULL,
	"status" integer NOT NULL
);
`

const m001Unapply = `
DROP TABLE IF EXISTS "solve_visit";
DROP TABLE IF EXISTS "solve_contest_problem_event";
DROP TABLE IF EXISTS "solve_contest_problem";
DROP TABLE IF EXISTS "solve_solution_event";
DROP TABLE IF EXISTS "solve_solution";
DROP TABLE IF EXISTS "solve_problem_event";
DROP TABLE IF EXISTS "solve_problem";
DROP TABLE IF EXISTS "solve_contest_event";
DROP TABLE IF EXISTS "solve_contest";
DROP TABLE IF EXISTS "solve_session_event";
DROP TABLE IF EXISTS "solve_session";
DROP TABLE IF EXISTS "solve_user_event";
DROP TABLE IF EXISTS "solve_user";
DROP TABLE IF EXISTS "solve_account_role_event";
DROP TABLE IF EXISTS "solve_account_role";
DROP TABLE IF EXISTS "solve_account_event";
DROP TABLE IF EXISTS "solve_account";
DROP TABLE IF EXISTS "solve_role_edge_event";
DROP TABLE IF EXISTS "solve_role_edge";
DROP TABLE IF EXISTS "solve_role_event";
DROP TABLE IF EXISTS "solve_role";
DROP TABLE IF EXISTS "solve_task_event";
DROP TABLE IF EXISTS "solve_task";
`

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
	if _, err := tx.Exec(m001Apply); err != nil {
		return err
	}
	return m.createRoles(c, tx)
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

func (m *m001) Unapply(c *core.Core, tx *sql.Tx) error {
	_, err := tx.Exec(m001Unapply)
	return err
}

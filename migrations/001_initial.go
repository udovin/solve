package migrations

import (
	"database/sql"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

type m001 struct{}

func (m *m001) Name() string {
	return "001_initial"
}

const m001Apply = `
-- models.Task
CREATE TABLE "solve_task" (
	"id" integer PRIMARY KEY,
	"status" integer NOT NULL,
	"kind" integer NOT NULL,
	"config" blob,
	"state" blob,
	"expire_time" bigint
);
-- models.TaskEvent
CREATE TABLE "solve_task_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"status" integer NOT NULL,
	"kind" integer NOT NULL,
	"config" blob,
	"state" blob,
	"expire_time" bigint
);
-- models.Role
CREATE TABLE "solve_role"
(
	"id" integer PRIMARY KEY,
	"code" varchar(255) NOT NULL
);
-- models.RoleEvent
CREATE TABLE "solve_role_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"code" varchar(255) NOT NULL
);
-- models.RoleEdge
CREATE TABLE "solve_role_edge"
(
	"id" integer PRIMARY KEY,
	"role_id" integer NOT NULL,
	"child_id" integer NOT NULL
);
-- models.RoleEdgeEvent
CREATE TABLE "solve_role_edge_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"role_id" integer NOT NULL,
	"child_id" integer NOT NULL
);
-- models.Account
CREATE TABLE "solve_account"
(
    "id" integer PRIMARY KEY,
    "kind" integer NOT NULL
);
-- models.AccountEvent
CREATE TABLE "solve_account_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"kind" integer NOT NULL
);
-- models.AccountRole
CREATE TABLE "solve_account_role"
(
	"id" integer PRIMARY KEY,
	"account_id" integer NOT NULL,
	"role_id" integer NOT NULL
);
-- models.AccountRoleEvent
CREATE TABLE "solve_account_role_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"account_id" integer NOT NULL,
	"role_id" integer NOT NULL
);
-- models.Session
CREATE TABLE "solve_session"
(
	"id" integer PRIMARY KEY,
	"account_id" integer NOT NULL,
	"secret" VARCHAR(56) NOT NULL,
	"create_time" bigint NOT NULL,
	"expire_time" bigint NOT NULL
);
-- models.SessionEvent
CREATE TABLE "solve_session_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"account_id" integer NOT NULL,
	"secret" VARCHAR(56) NOT NULL,
	"create_time" bigint NOT NULL,
	"expire_time" bigint NOT NULL
);
-- models.User
CREATE TABLE "solve_user"
(
	"id" integer PRIMARY KEY,
	"account_id" integer NOT NULL,
	"login" integer NOT NULL,
	"password_hash" integer NOT NULL,
	"password_salt" TEXT NOT NULL
);
-- models.UserEvent
CREATE TABLE "solve_user_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"account_id" integer NOT NULL,
	"login" integer NOT NULL,
	"password_hash" integer NOT NULL,
	"password_salt" TEXT NOT NULL
);
-- models.UserField
CREATE TABLE "solve_user_field"
(
	"id" integer PRIMARY KEY,
	"user_id" integer NOT NULL,
	"kind" integer NOT NULL,
	"data" TEXT NOT NULL
);
-- models.UserFieldEvent
CREATE TABLE "solve_user_field_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"user_id" integer NOT NULL,
	"kind" integer NOT NULL,
	"data" TEXT NOT NULL
);
-- models.Contest
CREATE TABLE "solve_contest"
(
	"id" integer PRIMARY KEY,
	"owner_id" integer,
	"config" blob
);
-- models.ContestEvent
CREATE TABLE "solve_contest_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"owner_id" integer,
	"config" blob
);
-- models.Problem
CREATE TABLE "solve_problem"
(
	"id" integer PRIMARY KEY,
	"owner_id" integer,
	"config" blob
);
-- models.ProblemEvent
CREATE TABLE "solve_problem_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"owner_id" integer,
	"config" blob
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
	"contest_id" bigint NOT NULL,
	"problem_id" bigint NOT NULL,
	"code" VARCHAR(32) NOT NULL
);
-- models.ContestProblemEvent
CREATE TABLE "solve_contest_problem_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"contest_id" bigint NOT NULL,
	"problem_id" bigint NOT NULL,
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
DROP TABLE IF EXISTS "solve_user_field_event";
DROP TABLE IF EXISTS "solve_user_field";
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
	for _, role := range []string{
		models.LoginRole,
		models.LogoutRole,
		models.RegisterRole,
		models.StatusRole,
		models.ObserveRolesRole,
		models.CreateRoleRole,
		models.DeleteRoleRole,
		models.ObserveRoleRolesRole,
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
		models.GuestGroupRole,
		models.UserGroupRole,
	} {
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
	return nil
}

func (m *m001) Unapply(c *core.Core, tx *sql.Tx) error {
	_, err := tx.Exec(m001Unapply)
	return err
}

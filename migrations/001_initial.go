package migrations

import (
	"database/sql"
)

type m001 struct{}

func (m *m001) Name() string {
	return "001_initial"
}

const m001Apply = `
-- models.Action
CREATE TABLE "solve_action" (
	"id" integer PRIMARY KEY,
	"status" integer NOT NULL,
	"type" integer NOT NULL,
	"config" blob,
	"state" blob,
	"expire_time" bigint
);
-- models.ActionEvent
CREATE TABLE "solve_action_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"status" integer NOT NULL,
	"type" integer NOT NULL,
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
-- models.User
CREATE TABLE "solve_user"
(
	"id" integer PRIMARY KEY,
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
	"login" integer NOT NULL,
	"password_hash" integer NOT NULL,
	"password_salt" TEXT NOT NULL
);
-- models.UserRole
CREATE TABLE "solve_user_role"
(
	"id" integer PRIMARY KEY,
	"user_id" integer NOT NULL,
	"role_id" integer NOT NULL
);
-- models.UserRoleEvent
CREATE TABLE "solve_user_role_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"user_id" integer NOT NULL,
	"role_id" integer NOT NULL
);
-- models.UserField
CREATE TABLE "solve_user_field"
(
	"id" integer PRIMARY KEY,
	"user_id" integer NOT NULL,
	"type" integer NOT NULL,
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
	"type" integer NOT NULL,
	"data" TEXT NOT NULL
);
-- models.Session
CREATE TABLE "solve_session"
(
	"id" integer PRIMARY KEY,
	"user_id" integer NOT NULL,
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
	"user_id" integer NOT NULL,
	"secret" VARCHAR(56) NOT NULL,
	"create_time" bigint NOT NULL,
	"expire_time" bigint NOT NULL
);
-- models.Contest
CREATE TABLE "solve_contest"
(
	"id" integer PRIMARY KEY,
	"config" blob
);
-- models.ContestEvent
CREATE TABLE "solve_contest_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"config" blob
);
-- models.Problem
CREATE TABLE "solve_problem"
(
	"id" integer PRIMARY KEY,
	"config" blob
);
-- models.ProblemEvent
CREATE TABLE "solve_problem_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"config" blob
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
	"user_id" integer,
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
DROP TABLE IF EXISTS "solve_problem_event";
DROP TABLE IF EXISTS "solve_problem";
DROP TABLE IF EXISTS "solve_contest_event";
DROP TABLE IF EXISTS "solve_contest";
DROP TABLE IF EXISTS "solve_session_event";
DROP TABLE IF EXISTS "solve_session";
DROP TABLE IF EXISTS "solve_user_field_event";
DROP TABLE IF EXISTS "solve_user_field";
DROP TABLE IF EXISTS "solve_user_role_event";
DROP TABLE IF EXISTS "solve_user_role";
DROP TABLE IF EXISTS "solve_user_event";
DROP TABLE IF EXISTS "solve_user";
DROP TABLE IF EXISTS "solve_role_edge_event";
DROP TABLE IF EXISTS "solve_role_edge";
DROP TABLE IF EXISTS "solve_role_event";
DROP TABLE IF EXISTS "solve_role";
DROP TABLE IF EXISTS "solve_action_event";
DROP TABLE IF EXISTS "solve_action";
`

func (m *m001) Apply(c Core, tx *sql.Tx) error {
	_, err := tx.Exec(m001Apply)
	return err
}

func (m *m001) Unapply(c Core, tx *sql.Tx) error {
	_, err := tx.Exec(m001Unapply)
	return err
}

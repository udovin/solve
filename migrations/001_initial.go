package migrations

import (
	"database/sql"

	"github.com/udovin/solve/core"
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

func (m *m001) Apply(c *core.Core, tx *sql.Tx) error {
	_, err := tx.Exec(m001Apply)
	return err
}

func (m *m001) Unapply(c *core.Core, tx *sql.Tx) error {
	_, err := tx.Exec(m001Unapply)
	return err
}

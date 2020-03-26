-- models.Action
CREATE TABLE IF NOT EXISTS "solve_action" (
	"id" integer PRIMARY KEY,
	"status" integer NOT NULL,
	"type" integer NOT NULL,
	"config" blob,
	"state" blob,
	"expire_time" bigint
);

-- models.ActionEvent
CREATE TABLE IF NOT EXISTS "solve_action_event"
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
CREATE TABLE IF NOT EXISTS "solve_role"
(
	"id" integer PRIMARY KEY,
	"code" varchar(255) NOT NULL
);

-- models.RoleEvent
CREATE TABLE IF NOT EXISTS "solve_role_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"code" varchar(255) NOT NULL
);

-- models.RoleEdge
CREATE TABLE IF NOT EXISTS "solve_role_edge"
(
	"id" integer PRIMARY KEY,
	"role_id" integer NOT NULL,
	"child_id" integer NOT NULL
);

-- models.RoleEdgeEvent
CREATE TABLE IF NOT EXISTS "solve_role_edge_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"role_id" integer NOT NULL,
	"child_id" integer NOT NULL
);

-- models.User
CREATE TABLE IF NOT EXISTS "solve_user"
(
	"id" integer PRIMARY KEY,
	"login" integer NOT NULL,
	"password_hash" integer NOT NULL,
	"password_salt" TEXT NOT NULL
);

-- models.UserEvent
CREATE TABLE IF NOT EXISTS "solve_user_event"
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
CREATE TABLE IF NOT EXISTS "solve_user_role"
(
	"id" integer PRIMARY KEY,
	"user_id" integer NOT NULL,
	"role_id" integer NOT NULL
);

-- models.UserRoleEvent
CREATE TABLE IF NOT EXISTS "solve_user_role_event"
(
	"event_id" integer PRIMARY KEY,
	"event_type" int8 NOT NULL,
	"event_time" bigint NOT NULL,
	"id" integer NOT NULL,
	"user_id" integer NOT NULL,
	"role_id" integer NOT NULL
);

-- models.UserField
CREATE TABLE IF NOT EXISTS "solve_user_field"
(
	"id" integer PRIMARY KEY,
	"user_id" integer NOT NULL,
	"type" integer NOT NULL,
	"data" TEXT NOT NULL
);

-- models.UserFieldEvent
CREATE TABLE IF NOT EXISTS "solve_user_field_event"
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
CREATE TABLE IF NOT EXISTS "solve_visit"
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

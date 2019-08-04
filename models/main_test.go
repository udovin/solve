package models

import (
	"database/sql"
	"os"
	"testing"

	"github.com/udovin/solve/config"
)

var db *sql.DB

var createTables = []string{
	// User store
	`CREATE TABLE test_user
(
	id integer not null
		constraint test_user_pk
			primary key autoincrement,
	login varchar(255) not null,
	password_hash char(88) not null,
	password_salt char(24) not null,
	create_time bigint not null
)`,
	`CREATE TABLE test_user_change
(
	change_id integer not null
		constraint test_user_change_pk
			primary key autoincrement,
	change_type int8 not null,
	change_time bigint not null,
	id integer not null,
	login varchar(255) not null,
	password_hash char(88) not null,
	password_salt char(24) not null,
	create_time bigint not null
)`,
	// User field store
	`CREATE TABLE test_user_field
(
	id integer not null
		constraint test_user_option_pk
			primary key autoincrement,
	user_id integer not null
		references test_user,
	name varchar(255) not null,
	data text not null
)`,
	`CREATE TABLE test_user_field_change
(
	change_id integer not null
		constraint test_user_option_change_pk
			primary key autoincrement,
	change_type int8 not null,
	change_time bigint not null,
	id integer not null,
	user_id integer not null,
	name varchar(255) not null,
	data text not null
)`,
	// Session store
	`CREATE TABLE test_session
(
	id integer not null
		constraint test_session_pk
			primary key autoincrement,
	user_id integer not null
		references test_user,
	secret char(56) not null,
	create_time bigint not null,
	expire_time bigint not null
)`,
	`CREATE TABLE test_session_change
(
	change_id integer not null
		constraint test_session_change_pk
			primary key autoincrement,
	change_type int8 not null,
	change_time bigint not null,
	id integer not null,
	user_id integer not null,
	secret char(56) not null,
	create_time bigint not null,
	expire_time bigint not null
)`,
	// Problem store
	`CREATE TABLE test_problem
(
	id integer not null
		constraint test_problem_pk
			primary key autoincrement,
	owner_id integer not null
		references test_user,
	create_time bigint not null
)`,
	`CREATE TABLE test_problem_change
(
	change_id integer not null
		constraint test_problem_change_pk
			primary key autoincrement,
	change_type int8 not null,
	change_time bigint not null,
	id integer not null,
	owner_id integer not null,
	create_time bigint not null
)`,
	// Contest store
	`CREATE TABLE test_contest
(
	id integer not null
		constraint test_contest_pk
			primary key autoincrement,
	owner_id integer not null
		references test_user,
	create_time bigint not null
)`,
	`CREATE TABLE test_contest_change
(
	change_id integer not null
		constraint test_contest_change_pk
			primary key autoincrement,
	change_type int8 not null,
	change_time bigint not null,
	id integer not null,
	owner_id integer not null,
	create_time bigint not null
)`,
	// Fake store
	`CREATE TABLE "test_fake_change"
(
	"change_id" INTEGER PRIMARY KEY,
	"change_type" INT8,
	"change_time" BIGINT,
	"id" INTEGER,
	"value" VARCHAR(255)
)`,
}

var dropTables = []string{
	// User store
	`DROP TABLE "test_user"`,
	`DROP TABLE "test_user_change"`,
	// User field store
	`DROP TABLE "test_user_field"`,
	`DROP TABLE "test_user_field_change"`,
	// Session store
	`DROP TABLE "test_session"`,
	`DROP TABLE "test_session_change"`,
	// Problem store
	`DROP TABLE "test_problem"`,
	`DROP TABLE "test_problem_change"`,
	// Contest store
	`DROP TABLE "test_contest"`,
	`DROP TABLE "test_contest_change"`,
	// Fake store
	`DROP TABLE "test_fake_change"`,
}

func setup(tb testing.TB) {
	cfg := config.DatabaseConfig{
		Driver:  config.SQLiteDriver,
		Options: config.SQLiteOptions{Path: "?mode=memory"},
	}
	var err error
	db, err = cfg.CreateDB()
	if err != nil {
		os.Exit(1)
	}
	for _, query := range createTables {
		if _, err := db.Exec(query); err != nil {
			tb.Fatal(err)
		}
	}
}

func teardown(tb testing.TB) {
	for _, query := range dropTables {
		if _, err := db.Exec(query); err != nil {
			tb.Fatal(err)
		}
	}
	_ = db.Close()
}

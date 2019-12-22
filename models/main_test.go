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
	create_time bigint not null,
	is_super boolean not null
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
	create_time bigint not null,
	is_super boolean not null
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
	// CurrentSession store
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
	user_id integer not null
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
	user_id integer not null,
	create_time bigint not null
)`,
	// Contest store
	`CREATE TABLE test_contest
(
	id integer not null
		constraint test_contest_pk
			primary key autoincrement,
	user_id integer not null
		references test_user,
	create_time bigint not null,
	title varchar(255) not null,
	config text not null
)`,
	`CREATE TABLE test_contest_change
(
	change_id integer not null
		constraint test_contest_change_pk
			primary key autoincrement,
	change_type int8 not null,
	change_time bigint not null,
	id integer not null,
	user_id integer not null,
	create_time bigint not null,
	title varchar(255) not null,
	config text not null
)`,
	// Solution store
	`CREATE TABLE test_solution
(
	id integer not null
		constraint test_contest_pk
			primary key autoincrement,
	user_id integer not null
		references test_user,
	problem_id integer not null
		references test_problem,
	contest_id integer
		references test_contest,
	compiler_id integer not null
		references test_compiler,
	source_code text not null,
	create_time bigint not null
)`,
	`CREATE TABLE test_solution_change
(
	change_id integer not null
		constraint test_contest_change_pk
			primary key autoincrement,
	change_type int8 not null,
	change_time bigint not null,
	id integer not null,
	user_id integer not null,
	problem_id integer not null,
	contest_id integer,
	compiler_id integer not null,
	source_code text not null,
	create_time bigint not null
)`,
	// Report store
	`CREATE TABLE test_report
(
	id integer not null
		constraint test_contest_pk
			primary key autoincrement,
	solution_id integer not null
		references test_solution,
	verdict int8 not null,
	data text not null,
	create_time bigint not null
)`,
	`CREATE TABLE test_report_change
(
	change_id integer not null
		constraint test_contest_change_pk
			primary key autoincrement,
	change_type int8 not null,
	change_time bigint not null,
	id integer not null,
	solution_id integer not null,
	verdict int8 not null,
	data text not null,
	create_time bigint not null
)`,
	// Participant store
	`CREATE TABLE test_participant
(
	id integer not null
		constraint test_contest_pk
			primary key autoincrement,
	type int8 not null,
	contest_id integer not null
		references test_contest,
	user_id integer not null
		references test_user,
	create_time bigint not null
)`,
	`CREATE TABLE test_participant_change
(
	change_id integer not null
		constraint test_contest_change_pk
			primary key autoincrement,
	change_type int8 not null,
	change_time bigint not null,
	id integer not null,
	type int8 not null,
	contest_id integer not null,
	user_id integer not null,
	create_time bigint not null
)`,
	// Participant store
	`CREATE TABLE test_contest_problem
(
	contest_id integer not null
		references test_contest,
	problem_id integer not null
		references test_problem,
	code varchar(255)
)`,
	`CREATE TABLE test_contest_problem_change
(
	change_id integer not null
		constraint test_contest_change_pk
			primary key autoincrement,
	change_type int8 not null,
	change_time bigint not null,
	contest_id integer not null,
	problem_id integer not null,
	code varchar(255)
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
	// CurrentSession store
	`DROP TABLE "test_session"`,
	`DROP TABLE "test_session_change"`,
	// Problem store
	`DROP TABLE "test_problem"`,
	`DROP TABLE "test_problem_change"`,
	// Contest store
	`DROP TABLE "test_contest"`,
	`DROP TABLE "test_contest_change"`,
	// Solution store
	`DROP TABLE "test_solution"`,
	`DROP TABLE "test_solution_change"`,
	// Report store
	`DROP TABLE "test_report"`,
	`DROP TABLE "test_report_change"`,
	// Participant store
	`DROP TABLE "test_participant"`,
	`DROP TABLE "test_participant_change"`,
	// Contest problem store
	`DROP TABLE "test_contest_problem"`,
	`DROP TABLE "test_contest_problem_change"`,
	// Fake store
	`DROP TABLE "test_fake_change"`,
}

func setup(tb testing.TB) {
	cfg := config.DB{
		Driver:  config.SQLiteDriver,
		Options: config.SQLiteOptions{Path: "?mode=memory"},
	}
	var err error
	db, err = cfg.Create()
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

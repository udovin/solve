package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/db"
)

type userManagerTest struct{}

func (t *userManagerTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "user" (` +
			`"id" integer PRIMARY KEY,` +
			`"login" varchar(64) NOT NULL,` +
			`"password_hash" varchar(255) NOT NULL,` +
			`"password_salt" varchar(255) NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "user_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"login" varchar(64) NOT NULL,` +
			`"password_hash" varchar(255) NOT NULL,` +
			`"password_salt" varchar(255) NOT NULL)`,
	)
	return err
}

func (t *userManagerTest) newManager() Manager {
	return NewUserManager("user", "user_event", "", db.SQLite)
}

func (t *userManagerTest) newObject() db.Object {
	return User{}
}

func (t *userManagerTest) createObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return m.(*UserManager).CreateTx(tx, o.(User))
}

func (t *userManagerTest) updateObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, m.(*UserManager).UpdateTx(tx, o.(User))
}

func (t *userManagerTest) deleteObject(
	m Manager, tx *sql.Tx, id int64,
) error {
	return m.(*UserManager).DeleteTx(tx, id)
}

func TestUserManager(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := managerTester{&userManagerTest{}}
	tester.Test(t)
}

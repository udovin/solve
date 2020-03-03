package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/db"
)

type roleManagerTest struct{}

func (t *roleManagerTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "role" (` +
			`"id" integer PRIMARY KEY,` +
			`"code" varchar(255) NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "role_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"code" varchar(255) NOT NULL)`,
	)
	return err
}

func (t *roleManagerTest) newManager() Manager {
	return NewRoleManager("role", "role_event", db.SQLite)
}

func (t *roleManagerTest) newObject() db.Object {
	return Role{}
}

func (t *roleManagerTest) createObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return m.(*RoleManager).CreateTx(tx, o.(Role))
}

func (t *roleManagerTest) updateObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, m.(*RoleManager).UpdateTx(tx, o.(Role))
}

func (t *roleManagerTest) deleteObject(
	m Manager, tx *sql.Tx, id int64,
) error {
	return m.(*RoleManager).DeleteTx(tx, id)
}

func TestRoleManager(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := managerTester{&roleManagerTest{}}
	tester.Test(t)
}

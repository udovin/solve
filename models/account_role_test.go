package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/db"
)

type accountRoleManagerTest struct{}

func (t *accountRoleManagerTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "account_role" (` +
			`"id" integer PRIMARY KEY,` +
			`"account_id" integer NOT NULL,` +
			`"role_id" integer NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "account_role_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"account_id" integer NOT NULL,` +
			`"role_id" integer NOT NULL)`,
	)
	return err
}

func (t *accountRoleManagerTest) newManager() Manager {
	return NewAccountRoleManager("account_role", "account_role_event", db.SQLite)
}

func (t *accountRoleManagerTest) newObject() db.Object {
	return AccountRole{}
}

func (t *accountRoleManagerTest) createObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return m.(*AccountRoleManager).CreateTx(tx, o.(AccountRole))
}

func (t *accountRoleManagerTest) updateObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, m.(*AccountRoleManager).UpdateTx(tx, o.(AccountRole))
}

func (t *accountRoleManagerTest) deleteObject(
	m Manager, tx *sql.Tx, id int64,
) error {
	return m.(*AccountRoleManager).DeleteTx(tx, id)
}

func TestUserRoleManager(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := managerTester{&accountRoleManagerTest{}}
	tester.Test(t)
}

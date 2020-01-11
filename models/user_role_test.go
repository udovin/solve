package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/db"
)

type userRoleManagerTest struct{}

func (t *userRoleManagerTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "user_role" (` +
			`"id" integer PRIMARY KEY,` +
			`"user_id" integer NOT NULL,` +
			`"role_id" integer NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "user_role_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"user_id" integer NOT NULL,` +
			`"role_id" integer NOT NULL)`,
	)
	return err
}

func (t *userRoleManagerTest) newManager() Manager {
	return NewUserRoleManager("user_role", "user_role_event", db.SQLite)
}

func (t *userRoleManagerTest) newObject() db.Object {
	return UserRole{}
}

func (t *userRoleManagerTest) createObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return m.(*UserRoleManager).CreateTx(tx, o.(UserRole))
}

func (t *userRoleManagerTest) updateObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, m.(*UserRoleManager).UpdateTx(tx, o.(UserRole))
}

func (t *userRoleManagerTest) deleteObject(
	m Manager, tx *sql.Tx, id int64,
) error {
	return m.(*UserRoleManager).DeleteTx(tx, id)
}

func TestUserRoleManager(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := managerTester{&userRoleManagerTest{}}
	tester.Test(t)
}

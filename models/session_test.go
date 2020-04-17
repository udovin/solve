package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/db"
)

type sessionManagerTest struct{}

func (t *sessionManagerTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "session" (` +
			`"id" integer PRIMARY KEY,` +
			`"account_id" integer NOT NULL,` +
			`"secret" varchar(255) NOT NULL,` +
			`"create_time" integer NOT NULL,` +
			`"expire_time" integer NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "session_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"account_id" integer NOT NULL,` +
			`"secret" varchar(255) NOT NULL,` +
			`"create_time" integer NOT NULL,` +
			`"expire_time" integer NOT NULL)`,
	)
	return err
}

func (t *sessionManagerTest) newManager() Manager {
	return NewSessionManager("session", "session_event", db.SQLite)
}

func (t *sessionManagerTest) newObject() db.Object {
	return Session{}
}

func (t *sessionManagerTest) createObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return m.(*SessionManager).CreateTx(tx, o.(Session))
}

func (t *sessionManagerTest) updateObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, m.(*SessionManager).UpdateTx(tx, o.(Session))
}

func (t *sessionManagerTest) deleteObject(
	m Manager, tx *sql.Tx, id int64,
) error {
	return m.(*SessionManager).DeleteTx(tx, id)
}

func TestSessionManager(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := managerTester{&sessionManagerTest{}}
	tester.Test(t)
}

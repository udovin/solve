package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/db"
)

type sessionStoreTest struct{}

func (t *sessionStoreTest) prepareDB(tx *sql.Tx) error {
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

func (t *sessionStoreTest) newStore() Store {
	return NewSessionStore("session", "session_event", db.SQLite)
}

func (t *sessionStoreTest) newObject() db.Object {
	return Session{}
}

func (t *sessionStoreTest) createObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return s.(*SessionStore).CreateTx(tx, o.(Session))
}

func (t *sessionStoreTest) updateObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, s.(*SessionStore).UpdateTx(tx, o.(Session))
}

func (t *sessionStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*SessionStore).DeleteTx(tx, id)
}

func TestSessionStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&sessionStoreTest{}}
	tester.Test(t)
}

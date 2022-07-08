package models

import (
	"database/sql"
	"testing"
)

type sessionStoreTest struct{}

func (t *sessionStoreTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "session" (` +
			`"id" integer PRIMARY KEY,` +
			`"account_id" integer NOT NULL,` +
			`"secret" varchar(255) NOT NULL,` +
			`"create_time" integer NOT NULL,` +
			`"expire_time" integer NOT NULL,` +
			`"remote_addr" varchar(255) NOT NULL,` +
			`"user_agent" varchar(255) NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "session_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_kind" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"event_account_id" integer NULL,` +
			`"id" integer NOT NULL,` +
			`"account_id" integer NOT NULL,` +
			`"secret" varchar(255) NOT NULL,` +
			`"create_time" integer NOT NULL,` +
			`"expire_time" integer NOT NULL,` +
			`"remote_addr" varchar(255) NOT NULL,` +
			`"user_agent" varchar(255) NOT NULL)`,
	)
	return err
}

func (t *sessionStoreTest) newStore() Store {
	return NewSessionStore(testDB, "session", "session_event")
}

func (t *sessionStoreTest) newObject() Object {
	return Session{}
}

func (t *sessionStoreTest) createObject(
	s Store, tx *sql.Tx, o Object,
) (Object, error) {
	object := o.(Session)
	err := s.(*SessionStore).Create(wrapContext(tx), &object)
	return object, err
}

func (t *sessionStoreTest) updateObject(
	s Store, tx *sql.Tx, o Object,
) (Object, error) {
	return o, s.(*SessionStore).Update(wrapContext(tx), o.(Session))
}

func (t *sessionStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*SessionStore).Delete(wrapContext(tx), id)
}

func TestSessionStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&sessionStoreTest{}}
	tester.Test(t)
}

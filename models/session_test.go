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

func (t *sessionStoreTest) newStore() CachedStore {
	return NewSessionStore(testDB, "session", "session_event")
}

func (t *sessionStoreTest) newObject() object {
	return Session{}
}

func (t *sessionStoreTest) createObject(
	s CachedStore, tx *sql.Tx, o object,
) (object, error) {
	object := o.(Session)
	err := s.(*SessionStore).Create(wrapContext(tx), &object)
	return object, err
}

func (t *sessionStoreTest) updateObject(
	s CachedStore, tx *sql.Tx, o object,
) (object, error) {
	return o, s.(*SessionStore).Update(wrapContext(tx), o.(Session))
}

func (t *sessionStoreTest) deleteObject(
	s CachedStore, tx *sql.Tx, id int64,
) error {
	return s.(*SessionStore).Delete(wrapContext(tx), id)
}

func TestSessionStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := CachedStoreTester{&sessionStoreTest{}}
	tester.Test(t)
}

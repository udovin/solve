package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/db"
)

type userFieldStoreTest struct{}

func (t *userFieldStoreTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "user_field" (` +
			`"id" integer PRIMARY KEY,` +
			`"user_id" integer NOT NULL,` +
			`"type" integer NOT NULL,` +
			`"data" text NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "user_field_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"user_id" integer NOT NULL,` +
			`"type" integer NOT NULL,` +
			`"data" text NOT NULL)`,
	)
	return err
}

func (t *userFieldStoreTest) newStore() Store {
	return NewUserFieldStore("user_field", "user_field_event", db.SQLite)
}

func (t *userFieldStoreTest) newObject() db.Object {
	return UserField{}
}

func (t *userFieldStoreTest) createObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return s.(*UserFieldStore).CreateTx(tx, o.(UserField))
}

func (t *userFieldStoreTest) updateObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, s.(*UserFieldStore).UpdateTx(tx, o.(UserField))
}

func (t *userFieldStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*UserFieldStore).DeleteTx(tx, id)
}

func TestUserFieldStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&userFieldStoreTest{}}
	tester.Test(t)
}

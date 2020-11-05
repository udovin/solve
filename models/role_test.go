package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/db"
)

type roleStoreTest struct{}

func (t *roleStoreTest) prepareDB(tx *sql.Tx) error {
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

func (t *roleStoreTest) newStore() Store {
	return NewRoleStore("role", "role_event", db.SQLite)
}

func (t *roleStoreTest) newObject() db.Object {
	return Role{}
}

func (t *roleStoreTest) createObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return s.(*RoleStore).CreateTx(tx, o.(Role))
}

func (t *roleStoreTest) updateObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, s.(*RoleStore).UpdateTx(tx, o.(Role))
}

func (t *roleStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*RoleStore).DeleteTx(tx, id)
}

func TestRoleStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&roleStoreTest{}}
	tester.Test(t)
}

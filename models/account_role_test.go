package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/db"
)

type accountRoleStoreTest struct{}

func (t *accountRoleStoreTest) prepareDB(tx *sql.Tx) error {
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

func (t *accountRoleStoreTest) newStore() Store {
	return NewAccountRoleStore("account_role", "account_role_event", db.SQLite)
}

func (t *accountRoleStoreTest) newObject() db.Object {
	return AccountRole{}
}

func (t *accountRoleStoreTest) createObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return s.(*AccountRoleStore).CreateTx(tx, o.(AccountRole))
}

func (t *accountRoleStoreTest) updateObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, s.(*AccountRoleStore).UpdateTx(tx, o.(AccountRole))
}

func (t *accountRoleStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*AccountRoleStore).DeleteTx(tx, id)
}

func TestUserRoleStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&accountRoleStoreTest{}}
	tester.Test(t)
}

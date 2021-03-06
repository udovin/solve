package models

import (
	"database/sql"
	"log"
	"testing"

	"github.com/udovin/solve/db"
)

type accountStoreTest struct{}

func (t *accountStoreTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "account" (` +
			`"id" integer PRIMARY KEY,` +
			`"kind" integer NOT NULL)`,
	); err != nil {
		log.Println("Error", err)
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "account_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"kind" integer NOT NULL)`,
	)
	log.Println("Error", err)
	return err
}

func (t *accountStoreTest) newStore() Store {
	return NewAccountStore("account", "account_event", db.SQLite)
}

func (t *accountStoreTest) newObject() db.Object {
	return Account{}
}

func (t *accountStoreTest) createObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return s.(*AccountStore).CreateTx(tx, o.(Account))
}

func (t *accountStoreTest) updateObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, s.(*AccountStore).UpdateTx(tx, o.(Account))
}

func (t *accountStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*AccountStore).DeleteTx(tx, id)
}

func TestAccountStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&accountStoreTest{}}
	tester.Test(t)
}

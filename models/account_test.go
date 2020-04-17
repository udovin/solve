package models

import (
	"database/sql"
	"log"
	"testing"

	"github.com/udovin/solve/db"
)

type accountManagerTest struct{}

func (t *accountManagerTest) prepareDB(tx *sql.Tx) error {
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

func (t *accountManagerTest) newManager() Manager {
	return NewAccountManager("account", "account_event", db.SQLite)
}

func (t *accountManagerTest) newObject() db.Object {
	return Account{}
}

func (t *accountManagerTest) createObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return m.(*AccountManager).CreateTx(tx, o.(Account))
}

func (t *accountManagerTest) updateObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, m.(*AccountManager).UpdateTx(tx, o.(Account))
}

func (t *accountManagerTest) deleteObject(
	m Manager, tx *sql.Tx, id int64,
) error {
	return m.(*AccountManager).DeleteTx(tx, id)
}

func TestAccountManager(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := managerTester{&accountManagerTest{}}
	tester.Test(t)
}

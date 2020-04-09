package models

import (
	"database/sql"
	"log"
	"testing"

	"github.com/udovin/solve/db"
)

type contestManagerTest struct{}

func (t *contestManagerTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "contest" (` +
			`"id" integer PRIMARY KEY,` +
			`"config" text NOT NULL)`,
	); err != nil {
		log.Println("Error", err)
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "contest_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"config" text NOT NULL)`,
	)
	log.Println("Error", err)
	return err
}

func (t *contestManagerTest) newManager() Manager {
	return NewContestManager("contest", "contest_event", db.SQLite)
}

func (t *contestManagerTest) newObject() db.Object {
	return Contest{}
}

func (t *contestManagerTest) createObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return m.(*ContestManager).CreateTx(tx, o.(Contest))
}

func (t *contestManagerTest) updateObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, m.(*ContestManager).UpdateTx(tx, o.(Contest))
}

func (t *contestManagerTest) deleteObject(
	m Manager, tx *sql.Tx, id int64,
) error {
	return m.(*ContestManager).DeleteTx(tx, id)
}

func TestContestManager(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := managerTester{&contestManagerTest{}}
	tester.Test(t)
}

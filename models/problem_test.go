package models

import (
	"database/sql"
	"log"
	"testing"

	"github.com/udovin/solve/db"
)

type problemStoreTest struct{}

func (t *problemStoreTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "problem" (` +
			`"id" integer PRIMARY KEY,` +
			`"config" text NOT NULL)`,
	); err != nil {
		log.Println("Error", err)
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "problem_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"config" text NOT NULL)`,
	)
	log.Println("Error", err)
	return err
}

func (t *problemStoreTest) newStore() Store {
	return NewProblemStore("problem", "problem_event", db.SQLite)
}

func (t *problemStoreTest) newObject() db.Object {
	return Problem{}
}

func (t *problemStoreTest) createObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return s.(*ProblemStore).CreateTx(tx, o.(Problem))
}

func (t *problemStoreTest) updateObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, s.(*ProblemStore).UpdateTx(tx, o.(Problem))
}

func (t *problemStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*ProblemStore).DeleteTx(tx, id)
}

func TestProblemStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&problemStoreTest{}}
	tester.Test(t)
}

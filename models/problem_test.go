package models

import (
	"database/sql"
	"log"
	"testing"

	"github.com/udovin/solve/db"
)

type problemManagerTest struct{}

func (t *problemManagerTest) prepareDB(tx *sql.Tx) error {
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

func (t *problemManagerTest) newManager() Manager {
	return NewProblemManager("problem", "problem_event", db.SQLite)
}

func (t *problemManagerTest) newObject() db.Object {
	return Problem{}
}

func (t *problemManagerTest) createObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return m.(*ProblemManager).CreateTx(tx, o.(Problem))
}

func (t *problemManagerTest) updateObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, m.(*ProblemManager).UpdateTx(tx, o.(Problem))
}

func (t *problemManagerTest) deleteObject(
	m Manager, tx *sql.Tx, id int64,
) error {
	return m.(*ProblemManager).DeleteTx(tx, id)
}

func TestProblemManager(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := managerTester{&problemManagerTest{}}
	tester.Test(t)
}

package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/db"
)

type contestProblemManagerTest struct{}

func (t *contestProblemManagerTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "contest_problem" (` +
			`"id" integer PRIMARY KEY,` +
			`"contest_id" integer NOT NULL,` +
			`"problem_id" integer NOT NULL,` +
			`"code" varchar(32) NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "contest_problem_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"contest_id" integer NOT NULL,` +
			`"problem_id" integer NOT NULL,` +
			`"code" varchar(32) NOT NULL)`,
	)
	return err
}

func (t *contestProblemManagerTest) newManager() Manager {
	return NewContestProblemManager("contest_problem", "contest_problem_event", db.SQLite)
}

func (t *contestProblemManagerTest) newObject() db.Object {
	return ContestProblem{}
}

func (t *contestProblemManagerTest) createObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return m.(*ContestProblemManager).CreateTx(tx, o.(ContestProblem))
}

func (t *contestProblemManagerTest) updateObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, m.(*ContestProblemManager).UpdateTx(tx, o.(ContestProblem))
}

func (t *contestProblemManagerTest) deleteObject(
	m Manager, tx *sql.Tx, id int64,
) error {
	return m.(*ContestProblemManager).DeleteTx(tx, id)
}

func TestContestProblemManager(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := managerTester{&contestProblemManagerTest{}}
	tester.Test(t)
}

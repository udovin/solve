package models

import (
	"database/sql"
	"testing"
)

type contestProblemStoreTest struct{}

func (t *contestProblemStoreTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "contest_problem" (` +
			`"id" integer PRIMARY KEY,` +
			`"contest_id" integer NOT NULL,` +
			`"problem_id" integer NOT NULL,` +
			`"code" varchar(32) NOT NULL,` +
			`"config" text NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "contest_problem_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_kind" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"event_account_id" integer NULL,` +
			`"id" integer NOT NULL,` +
			`"contest_id" integer NOT NULL,` +
			`"problem_id" integer NOT NULL,` +
			`"code" varchar(32) NOT NULL,` +
			`"config" text NOT NULL)`,
	)
	return err
}

func (t *contestProblemStoreTest) newStore() Store {
	return NewContestProblemStore(testDB, "contest_problem", "contest_problem_event")
}

func (t *contestProblemStoreTest) newObject() object {
	return ContestProblem{}
}

func (t *contestProblemStoreTest) createObject(
	s Store, tx *sql.Tx, o object,
) (object, error) {
	object := o.(ContestProblem)
	err := s.(*ContestProblemStore).Create(wrapContext(tx), &object)
	return object, err
}

func (t *contestProblemStoreTest) updateObject(
	s Store, tx *sql.Tx, o object,
) (object, error) {
	return o, s.(*ContestProblemStore).Update(wrapContext(tx), o.(ContestProblem))
}

func (t *contestProblemStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*ContestProblemStore).Delete(wrapContext(tx), id)
}

func TestContestProblemStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&contestProblemStoreTest{}}
	tester.Test(t)
}

package models

import (
	"database/sql"
	"log"
	"reflect"
	"testing"
)

type problemStoreTest struct{}

func (t *problemStoreTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "problem" (` +
			`"id" integer PRIMARY KEY,` +
			`"owner_id" integer,` +
			`"config" text NOT NULL,` +
			`"title" VARCHAR(255) NOT NULL)`,
	); err != nil {
		log.Println("Error", err)
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "problem_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"event_account_id" integer NULL,` +
			`"id" integer NOT NULL,` +
			`"owner_id" integer,` +
			`"config" text NOT NULL,` +
			`"title" VARCHAR(255) NOT NULL)`,
	)
	log.Println("Error", err)
	return err
}

func (t *problemStoreTest) newStore() Store {
	return NewProblemStore(testDB, "problem", "problem_event")
}

func (t *problemStoreTest) newObject() Object {
	return Problem{}
}

func (t *problemStoreTest) createObject(
	s Store, tx *sql.Tx, o Object,
) (Object, error) {
	problem := o.(Problem)
	if err := s.(*ProblemStore).Create(wrapContext(tx), &problem); err != nil {
		return Problem{}, err
	}
	return problem, nil
}

func (t *problemStoreTest) updateObject(
	s Store, tx *sql.Tx, o Object,
) (Object, error) {
	return o, s.(*ProblemStore).Update(wrapContext(tx), o.(Problem))
}

func (t *problemStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*ProblemStore).Delete(wrapContext(tx), id)
}

func TestProblemStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&problemStoreTest{}}
	tester.Test(t)
}

func TestProblemClone(t *testing.T) {
	problem := Problem{Config: JSON("{}")}
	problem.ID = 12345
	clone := problem.Clone()
	if !reflect.DeepEqual(problem, clone) {
		t.Fatalf("Problem clone is invalid, %v != %v", problem, clone)
	}
}

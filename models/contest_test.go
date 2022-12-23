package models

import (
	"database/sql"
	"log"
	"reflect"
	"testing"
)

type contestStoreTest struct{}

func (t *contestStoreTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "contest" (` +
			`"id" integer PRIMARY KEY,` +
			`"owner_id" integer,` +
			`"config" text NOT NULL,` +
			`"title" VARCHAR(255) NOT NULL)`,
	); err != nil {
		log.Println("Error", err)
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "contest_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_kind" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"event_account_id" integer NULL,` +
			`"id" integer NOT NULL,` +
			`"owner_id" integer,` +
			`"config" text NOT NULL,` +
			`"title" VARCHAR(255) NOT NULL)`,
	)
	return err
}

func (t *contestStoreTest) newStore() Store {
	return NewContestStore(testDB, "contest", "contest_event")
}

func (t *contestStoreTest) newObject() object {
	return Contest{}
}

func (t *contestStoreTest) createObject(
	s Store, tx *sql.Tx, o object,
) (object, error) {
	contest := o.(Contest)
	err := s.(*ContestStore).Create(wrapContext(tx), &contest)
	return contest, err
}

func (t *contestStoreTest) updateObject(
	s Store, tx *sql.Tx, o object,
) (object, error) {
	return o, s.(*ContestStore).Update(wrapContext(tx), o.(Contest))
}

func (t *contestStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*ContestStore).Delete(wrapContext(tx), id)
}

func TestContestStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&contestStoreTest{}}
	tester.Test(t)
}

func TestContestClone(t *testing.T) {
	contest := Contest{Config: JSON("{}")}
	contest.ID = 12345
	clone := contest.Clone()
	if !reflect.DeepEqual(contest, clone) {
		t.Fatalf("Contest clone is invalid, %v != %v", contest, clone)
	}
}

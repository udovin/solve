package models

import (
	"database/sql"
	"log"
	"reflect"
	"testing"

	"github.com/udovin/solve/db"
)

type contestStoreTest struct{}

func (t *contestStoreTest) prepareDB(tx *sql.Tx) error {
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
	return err
}

func (t *contestStoreTest) newStore() Store {
	return NewContestStore("contest", "contest_event", db.SQLite)
}

func (t *contestStoreTest) newObject() db.Object {
	return Contest{}
}

func (t *contestStoreTest) createObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return s.(*ContestStore).CreateTx(tx, o.(Contest))
}

func (t *contestStoreTest) updateObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, s.(*ContestStore).UpdateTx(tx, o.(Contest))
}

func (t *contestStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*ContestStore).DeleteTx(tx, id)
}

func TestContestStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&contestStoreTest{}}
	tester.Test(t)
}

func TestContestClone(t *testing.T) {
	contest := Contest{ID: 12345, Config: JSON("{}")}
	clone := contest.clone()
	if !reflect.DeepEqual(contest, clone) {
		t.Fatalf("Contest clone is invalid, %v != %v", contest, clone)
	}
}

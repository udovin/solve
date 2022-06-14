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
			`"event_account_id" integer NULL,` +
			`"id" integer NOT NULL,` +
			`"kind" integer NOT NULL)`,
	)
	log.Println("Error", err)
	return err
}

func (t *accountStoreTest) newStore() Store {
	return NewAccountStore(testDB, "account", "account_event")
}

func (t *accountStoreTest) newObject() db.Object {
	return Account{}
}

func (t *accountStoreTest) createObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	account := o.(Account)
	if err := s.(*AccountStore).Create(wrapContext(tx), &account); err != nil {
		return Account{}, err
	}
	return account, nil
}

func (t *accountStoreTest) updateObject(
	s Store, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, s.(*AccountStore).Update(wrapContext(tx), o.(Account))
}

func (t *accountStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*AccountStore).Delete(wrapContext(tx), id)
}

func TestAccountStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&accountStoreTest{}}
	tester.Test(t)
}

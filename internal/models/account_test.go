package models

import (
	"database/sql"
	"log"
	"testing"
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
			`"event_kind" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"event_account_id" integer NULL,` +
			`"id" integer NOT NULL,` +
			`"kind" integer NOT NULL)`,
	)
	log.Println("Error", err)
	return err
}

func (t *accountStoreTest) newStore() CachedStore {
	return NewAccountStore(testDB, "account", "account_event")
}

func (t *accountStoreTest) newObject() object {
	return Account{}
}

func (t *accountStoreTest) createObject(
	s CachedStore, tx *sql.Tx, o object,
) (object, error) {
	account := o.(Account)
	if err := s.(*AccountStore).Create(wrapContext(tx), &account); err != nil {
		return Account{}, err
	}
	return account, nil
}

func (t *accountStoreTest) updateObject(
	s CachedStore, tx *sql.Tx, o object,
) (object, error) {
	return o, s.(*AccountStore).Update(wrapContext(tx), o.(Account))
}

func (t *accountStoreTest) deleteObject(
	s CachedStore, tx *sql.Tx, id int64,
) error {
	return s.(*AccountStore).Delete(wrapContext(tx), id)
}

func TestAccountStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := CachedStoreTester{&accountStoreTest{}}
	tester.Test(t)
}

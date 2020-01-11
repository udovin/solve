package models

import (
	"database/sql"
	"testing"

	"github.com/udovin/solve/db"
)

type userFieldManagerTest struct{}

func (t *userFieldManagerTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "user_field" (` +
			`"id" integer PRIMARY KEY,` +
			`"user_id" integer NOT NULL,` +
			`"type" varchar(255) NOT NULL,` +
			`"data" text NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "user_field_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_type" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"id" integer NOT NULL,` +
			`"user_id" integer NOT NULL,` +
			`"type" varchar(255) NOT NULL,` +
			`"data" text NOT NULL)`,
	)
	return err
}

func (t *userFieldManagerTest) newManager() Manager {
	return NewUserFieldManager("user_field", "user_field_event", db.SQLite)
}

func (t *userFieldManagerTest) newObject() db.Object {
	return UserField{}
}

func (t *userFieldManagerTest) createObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return m.(*UserFieldManager).CreateTx(tx, o.(UserField))
}

func (t *userFieldManagerTest) updateObject(
	m Manager, tx *sql.Tx, o db.Object,
) (db.Object, error) {
	return o, m.(*UserFieldManager).UpdateTx(tx, o.(UserField))
}

func (t *userFieldManagerTest) deleteObject(
	m Manager, tx *sql.Tx, id int64,
) error {
	return m.(*UserFieldManager).DeleteTx(tx, id)
}

func TestUserFieldManager(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := managerTester{&userFieldManagerTest{}}
	tester.Test(t)
}

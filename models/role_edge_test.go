package models

import (
	"database/sql"
	"testing"
)

type roleEdgeStoreTest struct{}

func (t *roleEdgeStoreTest) prepareDB(tx *sql.Tx) error {
	if _, err := tx.Exec(
		`CREATE TABLE "role_edge" (` +
			`"id" integer PRIMARY KEY,` +
			`"role_id" integer NOT NULL,` +
			`"child_id" integer NOT NULL)`,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`CREATE TABLE "role_edge_event" (` +
			`"event_id" integer PRIMARY KEY,` +
			`"event_kind" int8 NOT NULL,` +
			`"event_time" bigint NOT NULL,` +
			`"event_account_id" integer NULL,` +
			`"id" integer NOT NULL,` +
			`"role_id" integer NOT NULL,` +
			`"child_id" integer NOT NULL)`,
	)
	return err
}

func (t *roleEdgeStoreTest) newStore() Store {
	return NewRoleEdgeStore(testDB, "role_edge", "role_edge_event")
}

func (t *roleEdgeStoreTest) newObject() Object {
	return RoleEdge{}
}

func (t *roleEdgeStoreTest) createObject(
	s Store, tx *sql.Tx, o Object,
) (Object, error) {
	object := o.(RoleEdge)
	err := s.(*RoleEdgeStore).Create(wrapContext(tx), &object)
	return object, err
}

func (t *roleEdgeStoreTest) updateObject(
	s Store, tx *sql.Tx, o Object,
) (Object, error) {
	return o, s.(*RoleEdgeStore).Update(wrapContext(tx), o.(RoleEdge))
}

func (t *roleEdgeStoreTest) deleteObject(
	s Store, tx *sql.Tx, id int64,
) error {
	return s.(*RoleEdgeStore).Delete(wrapContext(tx), id)
}

func TestRoleEdgeStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	tester := StoreTester{&roleEdgeStoreTest{}}
	tester.Test(t)
}

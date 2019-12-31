package db

import (
	"reflect"
	"testing"
)

type testExtraObject struct {
	A string `db:"a"`
	B int    `db:"b"`
}

type testObject struct {
	testExtraObject
	ID int64 `db:"id"`
	C  int   `db:"c"`
}

func (o testObject) ObjectID() int64 {
	return o.ID
}

func TestObjectStore(t *testing.T) {
	setup(t)
	defer teardown(t)
	store := NewObjectStore(testObject{}, "test_object", "id", SQLite)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Commit() }()
	objects := []testObject{
		{C: 8}, {C: 16}, {C: 5}, {C: 3},
		{testExtraObject: testExtraObject{A: "qwerty"}, C: 10},
	}
	for i, object := range objects {
		created, err := store.CreateObject(tx, object)
		if err != nil {
			t.Fatal(err)
		}
		objects[i].ID = created.ObjectID()
		if objects[i].ID != int64(i+1) {
			t.Fatal()
		}
		if objects[i] != created.(testObject) {
			t.Fatal()
		}
	}
	rows, err := store.LoadObjects(tx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var createdObjects []testObject
	for rows.Next() {
		createdObjects = append(createdObjects, rows.Object().(testObject))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(createdObjects, objects) {
		t.Fatal()
	}
}

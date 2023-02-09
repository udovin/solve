package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/config"
	"github.com/udovin/solve/db"
)

var testDB *gosql.DB

func testSetup(tb testing.TB) {
	cfg := config.DB{
		Options: config.SQLiteOptions{Path: ":memory:"},
	}
	var err error
	testDB, err = cfg.Create()
	if err != nil {
		os.Exit(1)
	}
}

func testTeardown(tb testing.TB) {
	_ = testDB.Close()
}

func TestEventKind(t *testing.T) {
	if s := fmt.Sprint(CreateEvent); s != "create" {
		t.Errorf("Expected %q, got %q", "create", s)
	}
	if s := fmt.Sprint(UpdateEvent); s != "update" {
		t.Errorf("Expected %q, got %q", "update", s)
	}
	if s := fmt.Sprint(DeleteEvent); s != "delete" {
		t.Errorf("Expected %q, got %q", "delete", s)
	}
	if s := fmt.Sprint(EventKind(-1)); s != "EventKind(-1)" {
		t.Errorf("Expected %q, got %q", "EventKind(-1)", s)
	}
}

type testObjectBase struct {
	String string `db:"string"`
	Int    int    `db:"int"`
	UInt   uint   `db:"uint"`
	Bool   bool   `db:"bool"`
	Bytes  []byte `db:"bytes"`
}

type testObject struct {
	testObjectBase
	ID   int64 `db:"id"`
	JSON JSON  `db:"json"`
}

func (o testObject) Clone() testObject {
	o.JSON = o.JSON.Clone()
	return o
}

func (o testObject) ObjectID() int64 {
	return o.ID
}

func (o *testObject) SetObjectID(id int64) {
	o.ID = id
}

type testObjectEvent struct {
	baseEvent
	testObject
}

func (e testObjectEvent) Object() testObject {
	return e.testObject
}

func (e *testObjectEvent) SetObject(o testObject) {
	e.testObject = o
}

type testStore struct {
	cachedStore[testObject, testObjectEvent, *testObject, *testObjectEvent]
	table, eventTable string
	objects           map[int64]testObject
}

func (s *testStore) Get(id int64) (testObject, error) {
	if object, ok := s.objects[id]; ok {
		return object, nil
	}
	return testObject{}, sql.ErrNoRows
}

//lint:ignore U1000 Used in generic interface.
func (s *testStore) reset() {
	s.objects = map[int64]testObject{}
}

//lint:ignore U1000 Used in generic interface.
func (s *testStore) onCreateObject(object testObject) {
	if _, ok := s.objects[object.ID]; ok {
		panic("object already exists")
	}
	s.objects[object.ID] = object
}

//lint:ignore U1000 Used in generic interface.
func (s *testStore) onDeleteObject(id int64) {
	object, ok := s.objects[id]
	if !ok {
		panic("object not found")
	}
	delete(s.objects, object.ID)
}

func migrateTestStore(t testing.TB, s *testStore) {
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal("Error:", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err := tx.Exec(fmt.Sprintf(
		`CREATE TABLE %q (`+
			`"id" integer PRIMARY KEY,`+
			`"string" varchar(255) NOT NULL,`+
			`"int" integer NOT NULL,`+
			`"uint" integer NOT NULL,`+
			`"bool" boolean NOT NULL,`+
			`"bytes" blob,`+
			`"json" blob NOT NULL)`,
		s.table,
	)); err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := tx.Exec(fmt.Sprintf(
		`CREATE TABLE %q (`+
			`"event_id" integer PRIMARY KEY,`+
			`"event_kind" int8 NOT NULL,`+
			`"event_time" bigint NOT NULL,`+
			`"event_account_id" integer NULL,`+
			`"id" integer NOT NULL,`+
			`"string" varchar(255) NOT NULL,`+
			`"int" integer NOT NULL,`+
			`"uint" integer NOT NULL,`+
			`"bool" boolean NOT NULL,`+
			`"bytes" blob,`+
			`"json" blob NOT NULL)`,
		s.eventTable,
	)); err != nil {
		t.Fatal("Error:", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal("Error:", err)
	}
}

func newTestStore() *testStore {
	impl := &testStore{
		table:      "test_object",
		eventTable: "test_object_event",
	}
	impl.cachedStore = makeCachedStore[testObject, testObjectEvent](
		testDB, impl.table, impl.eventTable, impl,
	)
	return impl
}

func testInitStore(t testing.TB, m CachedStore) {
	if err := m.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func testSyncStore(t testing.TB, s CachedStore) {
	if err := s.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func createTestObject(t testing.TB, s *testStore, o testObject) testObject {
	if err := s.Create(context.Background(), &o); err != nil {
		t.Fatal("Error:", err)
	}
	return o
}

func wrapContext(tx gosql.Runner) context.Context {
	ctx := context.Background()
	if v, ok := tx.(*sql.Tx); ok {
		ctx = db.WithTx(ctx, v)
	}
	return ctx
}

func updateTestObject(
	t testing.TB, s *testStore, o testObject, expErr error,
) {
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err = s.Update(wrapContext(tx), o); err != expErr {
		t.Fatalf("Expected %v, got %v", expErr, err)
	}
	if err == nil {
		if err := tx.Commit(); err != nil {
			t.Fatal("Error:", err)
		}
	}
}

func deleteTestObject(
	t testing.TB, s *testStore, id int64, expErr error,
) {
	if err := s.Delete(context.Background(), id); err != expErr {
		t.Fatal(err)
	}
}

func TestMakeCachedStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	master := newTestStore()
	replica := newTestStore()
	migrateTestStore(t, master)
	testInitStore(t, master)
	testInitStore(t, replica)
	object := testObject{
		testObjectBase: testObjectBase{
			String: "Test",
			Int:    822,
			UInt:   232,
			Bool:   true,
			Bytes:  []byte{8, 1, 4, 8},
		},
		JSON: JSON("\"Test message\""),
	}
	savedObject := createTestObject(t, master, object)
	if object.ID == savedObject.ID {
		t.Fatalf("IDs should be different: %v", object.ID)
	}
	if _, err := replica.Get(savedObject.ID); err != sql.ErrNoRows {
		t.Fatalf(
			"Replica already contains object: %v", savedObject.ID,
		)
	}
	checkReplicaObject := func(object testObject, expErr error) {
		testSyncStore(t, replica)
		loaded, err := replica.Get(object.ID)
		if err != expErr {
			t.Fatalf(
				"Replica does not contain object: %v", object.ID,
			)
		}
		if err == nil {
			if !reflect.DeepEqual(loaded, object) {
				t.Fatalf(
					"Objects are not equal: %v != %v", loaded, object,
				)
			}
		}
	}
	checkReplicaObject(savedObject, nil)
	savedObject.Int = 12345
	savedObject.JSON = JSON("\"Updated message\"")
	updateTestObject(t, master, savedObject, nil)
	checkReplicaObject(savedObject, nil)
	updateTestObject(t, master, testObject{ID: 100}, sql.ErrNoRows)
	deleteTestObject(t, master, savedObject.ID, nil)
	deleteTestObject(t, master, savedObject.ID, sql.ErrNoRows)
	checkReplicaObject(savedObject, sql.ErrNoRows)
}

func TestBaseStore_lockStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	store := newTestStore()
	migrateTestStore(t, store)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal("Error:", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err := store.lockStore(tx); err != nil {
		t.Fatal("Error:", err)
	}
}

func TestBaseStore_consumeEvent(t *testing.T) {
	store := cachedStore[testObject, testObjectEvent, *testObject, *testObjectEvent]{}
	if err := store.consumeEvent(testObjectEvent{
		baseEvent: makeBaseEvent(-1),
	}); err == nil {
		t.Fatal("Expected error")
	}
}

func TestBaseStore_InitTx(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	store := &testStore{
		table:      "invalid_object",
		eventTable: "invalid_object_event",
	}
	store.cachedStore = makeCachedStore[testObject, testObjectEvent](
		testDB, store.table, store.eventTable, store,
	)
	if err := store.Init(context.Background()); err == nil {
		t.Fatal("Expected error")
	}
}

func TestBaseEvent(t *testing.T) {
	ts := time.Now()
	event := baseEvent{BaseEventTime: ts.Unix()}
	if v := event.EventTime(); ts.Sub(v) > time.Second {
		t.Fatalf("Expected %v, got %v", ts, v)
	}
}

func TestNInt64_Value(t *testing.T) {
	var a NInt64
	va, err := a.Value()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if va != nil {
		t.Fatalf("Expected %v, got %v", nil, va)
	}
	var b NInt64 = 12345
	vb, err := b.Value()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if vb.(int64) != 12345 {
		t.Fatalf("Expected %v, got %v", 12345, vb)
	}
}

func TestNInt64_Scan(t *testing.T) {
	var a NInt64 = 12345
	if err := a.Scan(nil); err != nil {
		t.Fatal("Error:", err)
	}
	if a != 0 {
		t.Fatalf("Expected %v, got %v", 0, a)
	}
	var b NInt64
	if err := b.Scan(int64(12345)); err != nil {
		t.Fatal("Error:", err)
	}
	if b != 12345 {
		t.Fatalf("Expected %v, got %v", 12345, b)
	}
	var c NInt64
	if err := c.Scan(false); err == nil {
		t.Fatal("Expected error")
	}
}

// noinspection GoNilness
func TestJSON_Scan(t *testing.T) {
	var a JSON
	if err := a.Scan(nil); err != nil {
		t.Fatal("Error:", err)
	}
	if a != nil {
		t.Fatalf("Expected nil, but got: %v", a)
	}
	if err := a.Scan("null"); err != nil {
		t.Fatal("Error:", err)
	}
	if a != nil {
		t.Fatalf("Expected nil, but got: %v", a)
	}
	if err := a.Scan("{}"); err != nil {
		t.Fatal("Error:", err)
	}
	if a == nil {
		t.Fatalf("Unexpected nil")
	}
	if err := a.Scan("{"); err == nil {
		t.Fatal("Expected error")
	}
	if err := a.Scan([]byte("{}")); err != nil {
		t.Fatal("Error:", err)
	}
	if a == nil {
		t.Fatalf("Unexpected nil")
	}
	if err := a.Scan([]byte("{")); err == nil {
		t.Fatal("Expected error")
	}
	if err := a.Scan(cachedStore[testObject, testObjectEvent, *testObject, *testObjectEvent]{}); err == nil {
		t.Fatal("Expected error")
	}
}

func TestJSON_MarshalJSON(t *testing.T) {
	var a JSON
	if b, err := json.Marshal(a); err != nil {
		t.Fatal("Error:", err)
	} else if v := string(b); v != "null" {
		t.Fatalf("Expected %q, got: %q", "null", v)
	}
	a = JSON("{}")
	if b, err := json.Marshal(a); err != nil {
		t.Fatal("Error:", err)
	} else if v := string(b); v != "{}" {
		t.Fatalf("Expected %q, got: %q", "null", v)
	}
}

func TestJSON_UnmarshalJSON(t *testing.T) {
	var a JSON
	if err := json.Unmarshal([]byte("null"), &a); err != nil {
		t.Fatal("Error:", err)
	}
	if a != nil {
		t.Fatalf("Expected nil, got: %q", a)
	}
}

func TestJSON_clone(t *testing.T) {
	a := JSON(`{"hello": "world"}`)
	b := a.Clone()
	if string(a) != string(b) {
		t.Fatalf("Expected %s, got %s", a, b)
	}
	a[6] = '0'
	if string(a) == string(b) {
		t.Fatalf("Update should modify only one copy")
	}
	var c JSON = nil
	if d := c.Clone(); !reflect.DeepEqual(c, d) {
		t.Fatalf("Expected %v, got %v", c, d)
	}
}

type object interface {
	ObjectID() int64
}

type CachedStoreTestHelper interface {
	prepareDB(tx *sql.Tx) error
	newStore() CachedStore
	newObject() object
	createObject(s CachedStore, tx *sql.Tx, o object) (object, error)
	updateObject(s CachedStore, tx *sql.Tx, o object) (object, error)
	deleteObject(s CachedStore, tx *sql.Tx, id int64) error
}

func withTestTx(fn func(*sql.Tx) error) (err error) {
	var tx *sql.Tx
	if tx, err = testDB.Begin(); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()
	return fn(tx)
}

type CachedStoreTester struct {
	helper CachedStoreTestHelper
}

func (s *CachedStoreTester) Test(t testing.TB) {
	s.prepareDB(t)
	master := s.helper.newStore()
	if err := master.Init(context.Background()); err != nil {
		t.Fatal("Error:", err)
	}
	objects := s.createObjects(t, master)
	if err := master.Sync(context.Background()); err != nil {
		t.Fatal("Error:", err)
	}
	for _, object := range objects {
		if err := withTestTx(func(tx *sql.Tx) error {
			updated, err := s.helper.updateObject(master, tx, object)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(object, updated) {
				return fmt.Errorf("expected %v, got %v", object, updated)
			}
			return nil
		}); err != nil {
			t.Fatal("Error:", err)
		}
	}
	if err := master.Sync(context.Background()); err != nil {
		t.Fatal("Error:", err)
	}
	for _, object := range objects {
		if err := withTestTx(func(tx *sql.Tx) error {
			return s.helper.deleteObject(master, tx, object.ObjectID())
		}); err != nil {
			t.Fatal("Error:", err)
		}
	}
	if err := master.Sync(context.Background()); err != nil {
		t.Fatal("Error:", err)
	}
	for _, object := range objects {
		if err := withTestTx(func(tx *sql.Tx) error {
			return s.helper.deleteObject(master, tx, object.ObjectID())
		}); err != sql.ErrNoRows {
			t.Fatalf("Expected %q error, but got %q", sql.ErrNoRows, err)
		}
	}
	s.testFailedTx(t, master)
}

func (s *CachedStoreTester) createObjects(t testing.TB, mgr CachedStore) []object {
	var objects []object
	for i := 0; i < 100; i++ {
		object := s.helper.newObject()
		if err := withTestTx(func(tx *sql.Tx) error {
			created, err := s.helper.createObject(mgr, tx, object)
			if err != nil {
				return err
			}
			if id := created.ObjectID(); id <= 0 {
				return fmt.Errorf("object has invalid ID: %d", id)
			}
			objects = append(objects, created)
			return nil
		}); err != nil {
			t.Fatal("Error:", err)
		}
	}
	return objects
}

func (s *CachedStoreTester) testFailedTx(t testing.TB, mgr CachedStore) {
	if err := withTestTx(func(tx *sql.Tx) error {
		_ = tx.Rollback()
		_, err := s.helper.createObject(mgr, tx, s.helper.newObject())
		return err
	}); err == nil {
		t.Fatal("Expected error")
	}
}

func (s *CachedStoreTester) prepareDB(t testing.TB) {
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := s.helper.prepareDB(tx); err != nil {
		_ = tx.Rollback()
		t.Fatal("Error:", err)
	} else if err := tx.Commit(); err != nil {
		t.Fatal("Error:", err)
	}
}

func BenchmarkBaseStore_CreateTx(b *testing.B) {
	testSetup(b)
	defer testTeardown(b)
	store := newTestStore()
	migrateTestStore(b, store)
	if err := store.Init(context.Background()); err != nil {
		b.Fatal("Error: ", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytes, err := json.Marshal(i + 1)
		if err != nil {
			b.Fatal("Error: ", err)
		}
		obj := testObject{JSON: bytes}
		if err := store.Create(context.Background(), &obj); err != nil {
			b.Fatal("Error: ", err)
		}
		if err := store.Sync(context.Background()); err != nil {
			b.Fatal("Error: ", err)
		}
	}
}

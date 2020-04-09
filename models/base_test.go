package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/udovin/solve/config"
	"github.com/udovin/solve/db"
)

var testDB *sql.DB

func testSetup(tb testing.TB) {
	cfg := config.DB{
		Driver:  config.SQLiteDriver,
		Options: config.SQLiteOptions{Path: "?mode=memory"},
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

func TestEventType(t *testing.T) {
	if s := fmt.Sprintf("%s", CreateEvent); s != "Create" {
		t.Errorf("Expected %q, got %q", "Create", s)
	}
	if s := fmt.Sprintf("%s", UpdateEvent); s != "Update" {
		t.Errorf("Expected %q, got %q", "Update", s)
	}
	if s := fmt.Sprintf("%s", DeleteEvent); s != "Delete" {
		t.Errorf("Expected %q, got %q", "Delete", s)
	}
	if s := fmt.Sprintf("%s", EventType(-1)); s != "EventType(-1)" {
		t.Errorf("Expected %q, got %q", "EventType(-1)", s)
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

func (o testObject) ObjectID() int64 {
	return o.ID
}

type testObjectEvent struct {
	baseEvent
	testObject
}

func (e testObjectEvent) Object() db.Object {
	return e.testObject
}

func (e testObjectEvent) WithObject(o db.Object) ObjectEvent {
	e.testObject = o.(testObject)
	return e
}

type testManager struct {
	baseManager
	table, eventTable string
	objects           map[int64]testObject
}

func (m *testManager) Get(id int64) (testObject, error) {
	if object, ok := m.objects[id]; ok {
		return object, nil
	}
	return testObject{}, sql.ErrNoRows
}

func (m *testManager) CreateTx(
	tx *sql.Tx, object testObject,
) (testObject, error) {
	event, err := m.createObjectEvent(tx, testObjectEvent{
		makeBaseEvent(CreateEvent),
		object,
	})
	if err != nil {
		return testObject{}, err
	}
	return event.Object().(testObject), nil
}

func (m *testManager) UpdateTx(tx *sql.Tx, object testObject) error {
	_, err := m.createObjectEvent(tx, testObjectEvent{
		makeBaseEvent(UpdateEvent),
		object,
	})
	return err
}

func (m *testManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, testObjectEvent{
		makeBaseEvent(DeleteEvent),
		testObject{ID: id},
	})
	return err
}

func (m *testManager) reset() {
	m.objects = map[int64]testObject{}
}

func (m *testManager) onCreateObject(o db.Object) {
	if _, ok := m.objects[o.ObjectID()]; ok {
		panic("object already exists")
	}
	m.objects[o.ObjectID()] = o.(testObject)
}

func (m *testManager) onUpdateObject(o db.Object) {
	if _, ok := m.objects[o.ObjectID()]; !ok {
		panic("object not found")
	}
	m.objects[o.ObjectID()] = o.(testObject)
}

func (m *testManager) onDeleteObject(o db.Object) {
	if _, ok := m.objects[o.ObjectID()]; !ok {
		panic("object not found")
	}
	delete(m.objects, o.ObjectID())
}

func migrateTestManager(t testing.TB, m *testManager) {
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
		m.table,
	)); err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := tx.Exec(fmt.Sprintf(
		`CREATE TABLE %q (`+
			`"event_id" integer PRIMARY KEY,`+
			`"event_type" int8 NOT NULL,`+
			`"event_time" bigint NOT NULL,`+
			`"id" integer NOT NULL,`+
			`"string" varchar(255) NOT NULL,`+
			`"int" integer NOT NULL,`+
			`"uint" integer NOT NULL,`+
			`"bool" boolean NOT NULL,`+
			`"bytes" blob,`+
			`"json" blob NOT NULL)`,
		m.eventTable,
	)); err != nil {
		t.Fatal("Error:", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal("Error:", err)
	}
}

func newTestManager() *testManager {
	impl := &testManager{
		table:      "test_object",
		eventTable: "test_object_event",
	}
	impl.baseManager = makeBaseManager(
		testObject{}, impl.table,
		testObjectEvent{}, impl.eventTable,
		impl, db.SQLite,
	)
	return impl
}

func testInitManager(t testing.TB, m Manager) {
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err := m.InitTx(tx); err != nil {
		t.Fatal(err)
	}
}

func testSyncManager(t testing.TB, m Manager) {
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err := m.SyncTx(tx); err != nil {
		t.Fatal(err)
	}
}

func createTestObject(t testing.TB, m *testManager, o testObject) testObject {
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal("Error:", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if o, err = m.CreateTx(tx, o); err != nil {
		t.Fatal("Error:", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal("Error:", err)
	}
	return o
}

func updateTestObject(
	t testing.TB, m *testManager, o testObject, expErr error,
) {
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err = m.UpdateTx(tx, o); err != expErr {
		t.Fatalf("Expected %v, got %v", expErr, err)
	}
	if err == nil {
		if err := tx.Commit(); err != nil {
			t.Fatal("Error:", err)
		}
	}
}

func deleteTestObject(
	t testing.TB, m *testManager, id int64, expErr error,
) {
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	err = m.DeleteTx(tx, id)
	if err != expErr {
		t.Fatal(err)
	}
	if err == nil {
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMakeBaseManager(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	master := newTestManager()
	replica := newTestManager()
	migrateTestManager(t, master)
	testInitManager(t, master)
	testInitManager(t, replica)
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
		testSyncManager(t, replica)
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

func TestBaseManager_lockStore(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	manager := newTestManager()
	migrateTestManager(t, manager)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal("Error:", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err := manager.lockStore(tx); err != nil {
		t.Fatal("Error:", err)
	}
}

func TestBaseManager_consumeEvent(t *testing.T) {
	manager := baseManager{}
	if err := manager.consumeEvent(testObjectEvent{
		baseEvent: makeBaseEvent(-1),
	}); err == nil {
		t.Fatal("Expected error")
	}
}

func TestBaseManager_InitTx(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)
	manager := &testManager{
		table:      "invalid_object",
		eventTable: "invalid_object_event",
	}
	manager.baseManager = makeBaseManager(
		testObject{}, manager.table,
		testObjectEvent{}, manager.eventTable,
		manager, db.SQLite,
	)
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal("Error:", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err := manager.InitTx(tx); err == nil {
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
	var a NInt64 = 0
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

//noinspection GoNilness
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
	if err := a.Scan(baseManager{}); err == nil {
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
	b := a.clone()
	if string(a) != string(b) {
		t.Fatalf("Expected %s, got %s", a, b)
	}
	a[6] = '0'
	if string(a) == string(b) {
		t.Fatalf("Update should modify only one copy")
	}
}

type managerTestHelper interface {
	prepareDB(tx *sql.Tx) error
	newManager() Manager
	newObject() db.Object
	createObject(m Manager, tx *sql.Tx, o db.Object) (db.Object, error)
	updateObject(m Manager, tx *sql.Tx, o db.Object) (db.Object, error)
	deleteObject(m Manager, tx *sql.Tx, id int64) error
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

type managerTester struct {
	helper managerTestHelper
}

func (m *managerTester) Test(t testing.TB) {
	m.prepareDB(t)
	master := m.helper.newManager()
	if err := withTestTx(master.InitTx); err != nil {
		t.Fatal("Error:", err)
	}
	var objects []db.Object
	for i := 0; i < 100; i++ {
		object := m.helper.newObject()
		if err := withTestTx(func(tx *sql.Tx) error {
			created, err := m.helper.createObject(master, tx, object)
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
	if err := withTestTx(master.SyncTx); err != nil {
		t.Fatal("Error:", err)
	}
	for _, object := range objects {
		if err := withTestTx(func(tx *sql.Tx) error {
			updated, err := m.helper.updateObject(master, tx, object)
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
	if err := withTestTx(master.SyncTx); err != nil {
		t.Fatal("Error:", err)
	}
	for _, object := range objects {
		if err := withTestTx(func(tx *sql.Tx) error {
			return m.helper.deleteObject(master, tx, object.ObjectID())
		}); err != nil {
			t.Fatal("Error:", err)
		}
	}
	if err := withTestTx(master.SyncTx); err != nil {
		t.Fatal("Error:", err)
	}
	if err := withTestTx(func(tx *sql.Tx) error {
		_ = tx.Rollback()
		_, err := m.helper.createObject(master, tx, m.helper.newObject())
		return err
	}); err == nil {
		t.Fatal("Expected error")
	}
}

func (m *managerTester) prepareDB(t testing.TB) {
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := m.helper.prepareDB(tx); err != nil {
		_ = tx.Rollback()
		t.Fatal("Error:", err)
	} else if err := tx.Commit(); err != nil {
		t.Fatal("Error:", err)
	}
}

func BenchmarkBaseManager_CreateTx(b *testing.B) {
	testSetup(b)
	defer testTeardown(b)
	manager := newTestManager()
	migrateTestManager(b, manager)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := withTestTx(func(tx *sql.Tx) error {
			bytes, err := json.Marshal(i + 1)
			if err != nil {
				return err
			}
			_, err = manager.CreateTx(tx, testObject{
				JSON: JSON(bytes),
			})
			return err
		}); err != nil {
			b.Fatal("Error: ", err)
		}
	}
}

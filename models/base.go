package models

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sync"
	"time"

	"github.com/udovin/solve/db"
)

type indexInt64 map[int64]map[int64]struct{}

func (m indexInt64) Create(key, value int64) {
	if _, ok := m[key]; !ok {
		m[key] = map[int64]struct{}{}
	}
	m[key][value] = struct{}{}
}

func (m indexInt64) Delete(key, value int64) {
	delete(m[key], value)
	if len(m[key]) == 0 {
		delete(m, key)
	}
}

// NInt64 represents nullable int64 with zero value means null value.
type NInt64 int64

// Value returns value.
func (n NInt64) Value() (driver.Value, error) {
	if n == 0 {
		return nil, nil
	}
	return int64(n), nil
}

// Scan scans value.
func (n *NInt64) Scan(value interface{}) error {
	switch v := value.(type) {
	case nil:
		*n = 0
	case int64:
		*n = NInt64(v)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
	return nil
}

// EventType represents type of object event.
type EventType int8

const (
	// CreateEvent means that this is event of object creation.
	CreateEvent EventType = 1
	// DeleteEvent means that this is event of object deletion.
	DeleteEvent EventType = 2
	// UpdateEvent means that this is event of object modification.
	UpdateEvent EventType = 3
)

// String returns string representation of event.
func (t EventType) String() string {
	switch t {
	case CreateEvent:
		return "Create"
	case DeleteEvent:
		return "Delete"
	case UpdateEvent:
		return "Update"
	default:
		return fmt.Sprintf("EventType(%d)", t)
	}
}

// ObjectEvent represents event for object.
type ObjectEvent interface {
	db.Event
	// EventType should return type of object event.
	EventType() EventType
	// Object should return struct with object data.
	Object() db.Object
	// WithObject should return copy of event with replaced object.
	WithObject(db.Object) ObjectEvent
}

// baseEvent represents base for all events.
type baseEvent struct {
	// BaseEventID contains event id.
	BaseEventID int64 `db:"event_id" json:"EventID"`
	// BaseEventType contains type of event.
	BaseEventType EventType `db:"event_type" json:"EventType"`
	// BaseEventTime contains event type.
	BaseEventTime int64 `db:"event_time" json:"EventTime"`
}

// EventId returns id of this event.
func (e baseEvent) EventID() int64 {
	return e.BaseEventID
}

// EventTime returns time of this event.
func (e baseEvent) EventTime() time.Time {
	return time.Unix(e.BaseEventTime, 0)
}

// EventType returns type of this event.
func (e baseEvent) EventType() EventType {
	return e.BaseEventType
}

// makeBaseEvent creates baseEvent with specified type.
func makeBaseEvent(t EventType) baseEvent {
	return baseEvent{BaseEventType: t, BaseEventTime: time.Now().Unix()}
}

type baseManagerImpl interface {
	reset()
	addObject(o db.Object)
	onCreateObject(o db.Object)
	onDeleteObject(o db.Object)
	onUpdateObject(o db.Object)
}

// Manager represents store manager.
type Manager interface {
	InitTx(tx *sql.Tx) error
	SyncTx(tx *sql.Tx) error
}

type baseManager struct {
	table    string
	objects  db.ObjectStore
	events   db.EventStore
	consumer db.EventConsumer
	impl     baseManagerImpl
	dialect  db.Dialect
	mutex    sync.RWMutex
}

func (m *baseManager) InitTx(tx *sql.Tx) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	rows, err := m.objects.LoadObjects(tx)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()
	m.impl.reset()
	m.consumer = db.NewEventConsumer(m.events, 1)
	for rows.Next() {
		m.impl.addObject(rows.Object())
	}
	return rows.Err()
}

func (m *baseManager) SyncTx(tx *sql.Tx) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.consumer.ConsumeEvents(tx, m.consumeEvent)
}

func (m *baseManager) createObjectEvent(
	tx *sql.Tx, event ObjectEvent,
) (ObjectEvent, error) {
	switch object := event.Object(); event.EventType() {
	case CreateEvent:
		object, err := m.objects.CreateObject(tx, object)
		if err != nil {
			return nil, err
		}
		event = event.WithObject(object)
	case UpdateEvent:
		object, err := m.objects.UpdateObject(tx, object)
		if err != nil {
			return nil, err
		}
		event = event.WithObject(object)
	case DeleteEvent:
		if err := m.objects.DeleteObject(tx, object.ObjectID()); err != nil {
			return nil, err
		}
	}
	result, err := m.events.CreateEvent(tx, event)
	if err != nil {
		return nil, err
	}
	return result.(ObjectEvent), err
}

func (m *baseManager) lockStore(tx *sql.Tx) error {
	switch m.dialect {
	case db.SQLite:
		return nil
	default:
		_, err := tx.Exec(fmt.Sprintf("LOCK TABLE %q", m.table))
		return err
	}
}

func (m *baseManager) consumeEvent(e db.Event) error {
	switch v := e.(ObjectEvent); v.EventType() {
	case CreateEvent:
		m.impl.onCreateObject(v.Object())
	case DeleteEvent:
		m.impl.onDeleteObject(v.Object())
	case UpdateEvent:
		m.impl.onUpdateObject(v.Object())
	default:
		return fmt.Errorf("unexpected event type: %v", v.EventType())
	}
	return nil
}

func makeBaseManager(
	object db.Object, table string,
	event ObjectEvent, eventTable string,
	impl baseManagerImpl, dialect db.Dialect,
) baseManager {
	return baseManager{
		table:   table,
		objects: db.NewObjectStore(object, "id", table, dialect),
		events:  db.NewEventStore(event, "event_id", eventTable, dialect),
		impl:    impl,
		dialect: dialect,
	}
}

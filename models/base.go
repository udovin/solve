package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/udovin/gosql"
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
func (v NInt64) Value() (driver.Value, error) {
	if v == 0 {
		return nil, nil
	}
	return int64(v), nil
}

// Scan scans value.
func (v *NInt64) Scan(value interface{}) error {
	switch x := value.(type) {
	case nil:
		*v = 0
	case int64:
		*v = NInt64(x)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
	return nil
}

// JSON represents json value.
type JSON []byte

const nullJSON = "null"

// Value returns value.
func (v JSON) Value() (driver.Value, error) {
	if len(v) == 0 {
		return nullJSON, nil
	}
	return string(v), nil
}

// Scan scans value.
func (v *JSON) Scan(value interface{}) error {
	switch data := value.(type) {
	case nil:
		*v = nil
		return nil
	case []byte:
		return v.UnmarshalJSON(data)
	case string:
		return v.UnmarshalJSON([]byte(data))
	default:
		return fmt.Errorf("unsupported type: %T", data)
	}
}

// MarshalJSON marshals JSON.
func (v JSON) MarshalJSON() ([]byte, error) {
	if len(v) == 0 {
		return []byte(nullJSON), nil
	}
	return v, nil
}

// UnmarshalJSON unmarshals JSON.
func (v *JSON) UnmarshalJSON(bytes []byte) error {
	if !json.Valid(bytes) {
		return fmt.Errorf("invalid JSON value")
	}
	if string(bytes) == nullJSON {
		*v = nil
		return nil
	}
	*v = bytes
	return nil
}

func (v JSON) clone() JSON {
	if v == nil {
		return nil
	}
	c := make(JSON, len(v))
	copy(c, v)
	return c
}

// NString represents nullable string with empty value means null value.
type NString string

// Value returns value.
func (v NString) Value() (driver.Value, error) {
	if v == "" {
		return nil, nil
	}
	return string(v), nil
}

// Scan scans value.
func (v *NString) Scan(value interface{}) error {
	switch x := value.(type) {
	case nil:
		*v = ""
	case string:
		*v = NString(x)
	case []byte:
		*v = NString(x)
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
	BaseEventID int64 `db:"event_id"`
	// BaseEventType contains type of event.
	BaseEventType EventType `db:"event_type"`
	// BaseEventTime contains event type.
	BaseEventTime int64 `db:"event_time"`
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

type baseStoreImpl interface {
	reset()
	onCreateObject(o db.Object)
	onDeleteObject(o db.Object)
	onUpdateObject(o db.Object)
}

// Store represents cached store.
type Store interface {
	InitTx(tx gosql.WeakTx) error
	SyncTx(tx gosql.WeakTx) error
}

type baseStore struct {
	table    string
	objects  db.ObjectStore
	events   db.EventStore
	consumer db.EventConsumer
	impl     baseStoreImpl
	dialect  gosql.Dialect
	mutex    sync.RWMutex
}

func (s *baseStore) InitTx(tx gosql.WeakTx) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if err := s.initEvents(tx); err != nil {
		return err
	}
	return s.initObjects(tx)
}

const eventGapSkipWindow = 25000

func (s *baseStore) initEvents(tx gosql.WeakTx) error {
	beginID, err := s.events.LastEventID(tx)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		beginID = 1
	}
	if beginID > eventGapSkipWindow {
		beginID -= eventGapSkipWindow
	} else {
		beginID = 1
	}
	s.consumer = db.NewEventConsumer(s.events, beginID)
	return s.consumer.ConsumeEvents(tx, func(db.Event) error {
		return nil
	})
}

func (s *baseStore) initObjects(tx gosql.WeakTx) error {
	rows, err := s.objects.LoadObjects(tx)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()
	s.impl.reset()
	for rows.Next() {
		s.impl.onCreateObject(rows.Object())
	}
	return rows.Err()
}

func (s *baseStore) SyncTx(tx gosql.WeakTx) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.consumer.ConsumeEvents(tx, s.consumeEvent)
}

func (s *baseStore) createObjectEvent(
	tx gosql.WeakTx, event ObjectEvent,
) (ObjectEvent, error) {
	if err := gosql.WithEnsuredTx(tx, func(tx *sql.Tx) (err error) {
		event, err = s.createObjectEventTx(tx, event)
		return
	}); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *baseStore) createObjectEventTx(
	tx *sql.Tx, event ObjectEvent,
) (ObjectEvent, error) {
	switch object := event.Object(); event.EventType() {
	case CreateEvent:
		object, err := s.objects.CreateObject(tx, object)
		if err != nil {
			return nil, err
		}
		event = event.WithObject(object)
	case UpdateEvent:
		object, err := s.objects.UpdateObject(tx, object)
		if err != nil {
			return nil, err
		}
		event = event.WithObject(object)
	case DeleteEvent:
		if err := s.objects.DeleteObject(tx, object.ObjectID()); err != nil {
			return nil, err
		}
	}
	result, err := s.events.CreateEvent(tx, event)
	if err != nil {
		return nil, err
	}
	return result.(ObjectEvent), err
}

func (s *baseStore) lockStore(tx *sql.Tx) error {
	switch s.dialect {
	case gosql.SQLiteDialect:
		return nil
	default:
		_, err := tx.Exec(fmt.Sprintf("LOCK TABLE %q", s.table))
		return err
	}
}

func (s *baseStore) consumeEvent(e db.Event) error {
	switch v := e.(ObjectEvent); v.EventType() {
	case CreateEvent:
		s.impl.onCreateObject(v.Object())
	case DeleteEvent:
		s.impl.onDeleteObject(v.Object())
	case UpdateEvent:
		s.impl.onUpdateObject(v.Object())
	default:
		return fmt.Errorf("unexpected event type: %v", v.EventType())
	}
	return nil
}

func makeBaseStore(
	object db.Object, table string,
	event ObjectEvent, eventTable string,
	impl baseStoreImpl, dialect gosql.Dialect,
) baseStore {
	return baseStore{
		table:   table,
		objects: db.NewObjectStore(object, "id", table, dialect),
		events:  db.NewEventStore(event, "event_id", eventTable, dialect),
		impl:    impl,
		dialect: dialect,
	}
}

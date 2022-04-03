package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/db"
)

// Cloner represents object that can be cloned.
type Cloner[T any] interface {
	Clone() T
}

type index[K comparable] map[K]map[int64]struct{}

func makeIndex[K comparable]() index[K] {
	return map[K]map[int64]struct{}{}
}

func (m index[K]) Create(key K, id int64) {
	if _, ok := m[key]; !ok {
		m[key] = map[int64]struct{}{}
	}
	m[key][id] = struct{}{}
}

func (m index[K]) Delete(key K, id int64) {
	delete(m[key], id)
	if len(m[key]) == 0 {
		delete(m, key)
	}
}

type pairInt64 [2]int64

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
func (v *NInt64) Scan(value any) error {
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

var (
	_ driver.Valuer = NInt64(0)
	_ sql.Scanner   = (*NInt64)(nil)
)

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
func (v *JSON) Scan(value any) error {
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

func (v JSON) Clone() JSON {
	if v == nil {
		return nil
	}
	c := make(JSON, len(v))
	copy(c, v)
	return c
}

var (
	_ driver.Valuer = JSON{}
	_ sql.Scanner   = (*JSON)(nil)
)

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
func (v *NString) Scan(value any) error {
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

var (
	_ driver.Valuer = NString("")
	_ sql.Scanner   = (*NString)(nil)
)

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
type ObjectEvent[T db.Object] interface {
	db.Event
	// EventType should return type of object event.
	EventType() EventType
	// Object should return struct with object data.
	Object() T
	// WithObject should return copy of event with replaced object.
	WithObject(T) ObjectEvent[T]
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

type baseStoreImpl[T db.Object] interface {
	reset()
	makeObject(id int64) T
	makeObjectEvent(EventType) ObjectEvent[T]
	onCreateObject(T)
	onDeleteObject(T)
	onUpdateObject(T)
}

// Store represents cached store.
type Store interface {
	Init(ctx context.Context) error
	Sync(ctx context.Context) error
}

type baseStore[T db.Object, E ObjectEvent[T]] struct {
	db       *gosql.DB
	table    string
	objects  db.ObjectStore[T]
	events   db.EventStore[E]
	consumer db.EventConsumer[E]
	impl     baseStoreImpl[T]
	mutex    sync.RWMutex
}

// DB returns store database.
func (s *baseStore[T, E]) DB() *gosql.DB {
	return s.db
}

func (s *baseStore[T, E]) Init(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if err := s.initEvents(ctx); err != nil {
		return err
	}
	return s.initObjects(ctx)
}

const eventGapSkipWindow = 25000

func (s *baseStore[T, E]) initEvents(ctx context.Context) error {
	beginID, err := s.events.LastEventID(ctx)
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
	s.consumer = db.NewEventConsumer[E](s.events, beginID)
	return s.consumer.ConsumeEvents(ctx, func(E) error {
		return nil
	})
}

func (s *baseStore[T, E]) initObjects(ctx context.Context) error {
	rows, err := s.objects.LoadObjects(ctx)
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

func (s *baseStore[T, E]) Sync(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.consumer.ConsumeEvents(ctx, s.consumeEvent)
}

func wrapContext(tx gosql.WeakTx) context.Context {
	ctx := context.Background()
	if v, ok := tx.(*sql.Tx); ok {
		ctx = gosql.WithTx(ctx, v)
	}
	return ctx
}

// Create creates object and returns copy with valid ID.
func (s *baseStore[T, E]) Create(ctx context.Context, object *T) error {
	event := s.impl.makeObjectEvent(CreateEvent).WithObject(*object).(E)
	if err := s.createObjectEvent(ctx, &event); err != nil {
		return err
	}
	*object = event.Object()
	return nil
}

// Update updates object with specified ID.
func (s *baseStore[T, E]) Update(ctx context.Context, object T) error {
	event := s.impl.makeObjectEvent(UpdateEvent).WithObject(object).(E)
	return s.createObjectEvent(ctx, &event)
}

// Delete deletes compiler with specified ID.
func (s *baseStore[T, E]) Delete(ctx context.Context, id int64) error {
	object := s.impl.makeObject(id)
	event := s.impl.makeObjectEvent(DeleteEvent).WithObject(object).(E)
	return s.createObjectEvent(ctx, &event)
}

var sqlRepeatableRead = gosql.WithTxOptions(&sql.TxOptions{
	Isolation: sql.LevelRepeatableRead,
})

func (s *baseStore[T, E]) createObjectEvent(
	ctx context.Context, event *E,
) error {
	// Force creation of new transaction.
	if tx := gosql.GetTx(ctx); tx == nil {
		return gosql.WrapTx(s.db, func(tx *sql.Tx) error {
			return s.createObjectEvent(gosql.WithTx(ctx, tx), event)
		}, gosql.WithContext(ctx), sqlRepeatableRead)
	}
	switch object := (*event).Object(); (*event).EventType() {
	case CreateEvent:
		if err := s.objects.CreateObject(ctx, &object); err != nil {
			return err
		}
		*event = (*event).WithObject(object).(E)
	case UpdateEvent:
		if err := s.objects.UpdateObject(ctx, &object); err != nil {
			return err
		}
		*event = (*event).WithObject(object).(E)
	case DeleteEvent:
		if err := s.objects.DeleteObject(ctx, object.ObjectID()); err != nil {
			return err
		}
	}
	return s.events.CreateEvent(ctx, event)
}

func (s *baseStore[T, E]) lockStore(tx *sql.Tx) error {
	switch s.db.Dialect() {
	case gosql.SQLiteDialect:
		return nil
	default:
		_, err := tx.Exec(fmt.Sprintf("LOCK TABLE %q", s.table))
		return err
	}
}

func (s *baseStore[T, E]) consumeEvent(e E) error {
	switch e.EventType() {
	case CreateEvent:
		s.impl.onCreateObject(e.Object())
	case DeleteEvent:
		s.impl.onDeleteObject(e.Object())
	case UpdateEvent:
		s.impl.onUpdateObject(e.Object())
	default:
		return fmt.Errorf("unexpected event type: %v", e.EventType())
	}
	return nil
}

func makeBaseStore[T db.Object, E ObjectEvent[T]](
	dbConn *gosql.DB,
	table, eventTable string,
	impl baseStoreImpl[T],
) baseStore[T, E] {
	return baseStore[T, E]{
		db:      dbConn,
		table:   table,
		objects: db.NewObjectStore[T]("id", table, dbConn),
		events:  db.NewEventStore[E]("event_id", eventTable, dbConn),
		impl:    impl,
	}
}

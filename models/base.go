// Package models contains tools for working with solve objects stored
// in different databases like SQLite or Postgres.
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

type pair[F, S any] struct {
	First  F
	Second S
}

func makePair[F, S any](f F, s S) pair[F, S] {
	return pair[F, S]{First: f, Second: s}
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
		return "create"
	case DeleteEvent:
		return "delete"
	case UpdateEvent:
		return "update"
	default:
		return fmt.Sprintf("EventType(%d)", t)
	}
}

type ObjectEventPtr[T any, E any] interface {
	*E
	EventID() int64
	SetEventID(int64)
	EventTime() time.Time
	SetEventTime(time.Time)
	EventType() EventType
	SetEventType(EventType)
	SetEventAccountID(int64)
	Object() T
	SetObject(T)
	ObjectID() int64
	SetObjectID(int64)
}

// baseEvent represents base for all events.
type baseEvent struct {
	// BaseEventID contains event id.
	BaseEventID int64 `db:"event_id"`
	// BaseEventType contains type of event.
	BaseEventType EventType `db:"event_type"`
	// BaseEventTime contains event type.
	BaseEventTime int64 `db:"event_time"`
	// EventAccountID contains account id.
	EventAccountID NInt64 `db:"event_account_id"`
}

// EventID returns id of this event.
func (e baseEvent) EventID() int64 {
	return e.BaseEventID
}

// SetEventID updates id of this event.
func (e *baseEvent) SetEventID(id int64) {
	e.BaseEventID = id
}

// EventTime returns time of this event.
func (e baseEvent) EventTime() time.Time {
	return time.Unix(e.BaseEventTime, 0)
}

// SetEventTime updates time of this event.
func (e *baseEvent) SetEventTime(t time.Time) {
	e.BaseEventTime = t.Unix()
}

// EventType returns type of this event.
func (e baseEvent) EventType() EventType {
	return e.BaseEventType
}

// SetEventType updates type of this event.
func (e *baseEvent) SetEventType(typ EventType) {
	e.BaseEventType = typ
}

func (e *baseEvent) SetEventAccountID(accountID int64) {
	e.EventAccountID = NInt64(accountID)
}

type accountIDKey struct{}

func WithAccountID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, accountIDKey{}, id)
}

// GetAccountID returns account ID or zero if there is no account.
func GetAccountID(ctx context.Context) int64 {
	if id, ok := ctx.Value(accountIDKey{}).(int64); ok {
		return id
	}
	return 0
}

// makeBaseEvent creates baseEvent with specified type.
func makeBaseEvent(t EventType) baseEvent {
	return baseEvent{BaseEventType: t, BaseEventTime: time.Now().Unix()}
}

type baseStoreImpl[
	T any, E any, TPtr db.ObjectPtr[T], EPtr db.EventPtr[E],
] interface {
	reset()
	onCreateObject(T)
	onDeleteObject(int64)
	onUpdateObject(T)
}

// Store represents cached store.
type Store interface {
	Init(ctx context.Context) error
	Sync(ctx context.Context) error
}

type baseStore[
	T any, E any, TPtr db.ObjectPtr[T], EPtr ObjectEventPtr[T, E],
] struct {
	db       *gosql.DB
	table    string
	objects  db.ObjectStore[T, TPtr]
	events   db.EventStore[E, EPtr]
	consumer db.EventConsumer[E, EPtr]
	impl     baseStoreImpl[T, E, TPtr, EPtr]
	mutex    sync.RWMutex
}

// DB returns store database.
func (s *baseStore[T, E, TPtr, EPtr]) DB() *gosql.DB {
	return s.db
}

func (s *baseStore[T, E, TPtr, EPtr]) Init(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.initUnlocked(ctx)
}

func (s *baseStore[T, E, TPtr, EPtr]) initUnlocked(ctx context.Context) error {
	if tx := db.GetTx(ctx); tx == nil {
		return gosql.WrapTx(ctx, s.db, func(tx *sql.Tx) error {
			return s.initUnlocked(db.WithTx(ctx, tx))
		}, sqlReadOnly)
	}
	if err := s.initEvents(ctx); err != nil {
		return err
	}
	return s.initObjects(ctx)
}

const eventGapSkipWindow = 25000

func (s *baseStore[T, E, TPtr, EPtr]) initEvents(ctx context.Context) error {
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
	s.consumer = db.NewEventConsumer[E, EPtr](s.events, beginID)
	return s.consumer.ConsumeEvents(ctx, func(E) error {
		return nil
	})
}

func (s *baseStore[T, E, TPtr, EPtr]) initObjects(ctx context.Context) error {
	rows, err := s.objects.LoadObjects(ctx)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()
	s.impl.reset()
	for rows.Next() {
		s.impl.onCreateObject(rows.Row())
	}
	return rows.Err()
}

func (s *baseStore[T, E, TPtr, EPtr]) Sync(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.consumer.ConsumeEvents(ctx, s.consumeEvent)
}

func (s *baseStore[T, E, TPtr, EPtr]) newObjectEvent(ctx context.Context, kind EventType) EPtr {
	var event E
	var eventPtr EPtr = &event
	eventPtr.SetEventTime(time.Now())
	eventPtr.SetEventType(kind)
	eventPtr.SetEventAccountID(GetAccountID(ctx))
	return eventPtr
}

// Create creates object and returns copy with valid ID.
func (s *baseStore[T, E, TPtr, EPtr]) Create(ctx context.Context, object TPtr) error {
	eventPtr := s.newObjectEvent(ctx, CreateEvent)
	eventPtr.SetObject(*object)
	if err := s.createObjectEvent(ctx, eventPtr); err != nil {
		return err
	}
	*object = eventPtr.Object()
	return nil
}

// Update updates object with specified ID.
func (s *baseStore[T, E, TPtr, EPtr]) Update(ctx context.Context, object T) error {
	eventPtr := s.newObjectEvent(ctx, UpdateEvent)
	eventPtr.SetObject(object)
	return s.createObjectEvent(ctx, eventPtr)
}

// Delete deletes compiler with specified ID.
func (s *baseStore[T, E, TPtr, EPtr]) Delete(ctx context.Context, id int64) error {
	eventPtr := s.newObjectEvent(ctx, DeleteEvent)
	eventPtr.SetObjectID(id)
	return s.createObjectEvent(ctx, eventPtr)
}

var (
	sqlRepeatableRead = gosql.WithIsolation(sql.LevelRepeatableRead)
	sqlReadOnly       = gosql.WithReadOnly(true)
)

func (s *baseStore[T, E, TPtr, EPtr]) createObjectEvent(
	ctx context.Context, eventPtr EPtr,
) error {
	// Force creation of new transaction.
	if tx := db.GetTx(ctx); tx == nil {
		return gosql.WrapTx(ctx, s.db, func(tx *sql.Tx) error {
			return s.createObjectEvent(db.WithTx(ctx, tx), eventPtr)
		}, sqlRepeatableRead)
	}
	switch object := eventPtr.Object(); eventPtr.EventType() {
	case CreateEvent:
		if err := s.objects.CreateObject(ctx, &object); err != nil {
			return err
		}
		eventPtr.SetObject(object)
	case UpdateEvent:
		if err := s.objects.UpdateObject(ctx, &object); err != nil {
			return err
		}
		eventPtr.SetObject(object)
	case DeleteEvent:
		if err := s.objects.DeleteObject(ctx, eventPtr.ObjectID()); err != nil {
			return err
		}
	}
	return s.events.CreateEvent(ctx, eventPtr)
}

func (s *baseStore[T, E, TPtr, EPtr]) lockStore(tx *sql.Tx) error {
	switch s.db.Dialect() {
	case gosql.SQLiteDialect:
		return nil
	default:
		_, err := tx.Exec(fmt.Sprintf("LOCK TABLE %q", s.table))
		return err
	}
}

func (s *baseStore[T, E, TPtr, EPtr]) onUpdateObject(object T) {
	s.impl.onDeleteObject(TPtr(&object).ObjectID())
	s.impl.onCreateObject(object)
}

func (s *baseStore[T, E, TPtr, EPtr]) consumeEvent(event E) error {
	var eventPtr EPtr = &event
	switch object := eventPtr.Object(); eventPtr.EventType() {
	case CreateEvent:
		s.impl.onCreateObject(object)
	case DeleteEvent:
		s.impl.onDeleteObject(eventPtr.ObjectID())
	case UpdateEvent:
		s.impl.onUpdateObject(object)
	default:
		return fmt.Errorf("unexpected event type: %v", eventPtr.EventType())
	}
	return nil
}

func makeBaseStore[T any, E any, TPtr db.ObjectPtr[T], EPtr ObjectEventPtr[T, E]](
	conn *gosql.DB,
	table, eventTable string,
	impl baseStoreImpl[T, E, TPtr, EPtr],
) baseStore[T, E, TPtr, EPtr] {
	return baseStore[T, E, TPtr, EPtr]{
		db:      conn,
		table:   table,
		objects: db.NewObjectStore[T, TPtr]("id", table, conn),
		events:  db.NewEventStore[E, EPtr]("event_id", eventTable, conn),
		impl:    impl,
	}
}

// Package models contains tools for working with solve objects stored
// in different databases like SQLite or Postgres.
package models

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/db"
)

type index[K comparable] map[K]map[int64]struct{}

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

// EventKind represents kind of object event.
type EventKind int8

const (
	// CreateEvent means that this is event of object creation.
	CreateEvent EventKind = 1
	// DeleteEvent means that this is event of object deletion.
	DeleteEvent EventKind = 2
	// UpdateEvent means that this is event of object modification.
	UpdateEvent EventKind = 3
)

// String returns string representation of event.
func (t EventKind) String() string {
	switch t {
	case CreateEvent:
		return "create"
	case DeleteEvent:
		return "delete"
	case UpdateEvent:
		return "update"
	default:
		return fmt.Sprintf("EventKind(%d)", t)
	}
}

type ObjectEventPtr[T any, E any] interface {
	*E
	EventID() int64
	SetEventID(int64)
	EventTime() time.Time
	SetEventTime(time.Time)
	EventKind() EventKind
	SetEventKind(EventKind)
	SetEventAccountID(int64)
	Object() T
	SetObject(T)
	ObjectID() int64
	SetObjectID(int64)
}

// baseObject represents base for all objects.
type baseObject struct {
	// ID contains object id.
	ID int64 `db:"id"`
}

// ObjectID returns ID of object.
func (o baseObject) ObjectID() int64 {
	return o.ID
}

// SetObjectID updates ID of object.
func (o *baseObject) SetObjectID(id int64) {
	o.ID = id
}

// baseEvent represents base for all events.
type baseEvent struct {
	// BaseEventID contains event id.
	BaseEventID int64 `db:"event_id"`
	// BaseEventKind contains type of event.
	BaseEventKind EventKind `db:"event_kind"`
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

// EventKind returns type of this event.
func (e baseEvent) EventKind() EventKind {
	return e.BaseEventKind
}

// SetEventKind updates type of this event.
func (e *baseEvent) SetEventKind(typ EventKind) {
	e.BaseEventKind = typ
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
func makeBaseEvent(t EventKind) baseEvent {
	return baseEvent{BaseEventKind: t, BaseEventTime: time.Now().Unix()}
}

type baseStoreImpl[T any] interface {
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
	impl     baseStoreImpl[T]
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

func (s *baseStore[T, E, TPtr, EPtr]) newObjectEvent(ctx context.Context, kind EventKind) EPtr {
	var event E
	var eventPtr EPtr = &event
	eventPtr.SetEventTime(time.Now())
	eventPtr.SetEventKind(kind)
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
	switch object := eventPtr.Object(); eventPtr.EventKind() {
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
	switch object := eventPtr.Object(); eventPtr.EventKind() {
	case CreateEvent:
		s.impl.onCreateObject(object)
	case DeleteEvent:
		s.impl.onDeleteObject(eventPtr.ObjectID())
	case UpdateEvent:
		s.impl.onUpdateObject(object)
	default:
		return fmt.Errorf("unexpected event type: %v", eventPtr.EventKind())
	}
	return nil
}

func makeBaseStore[T any, E any, TPtr db.ObjectPtr[T], EPtr ObjectEventPtr[T, E]](
	conn *gosql.DB,
	table, eventTable string,
	impl baseStoreImpl[T],
) baseStore[T, E, TPtr, EPtr] {
	return baseStore[T, E, TPtr, EPtr]{
		db:      conn,
		table:   table,
		objects: db.NewObjectStore[T, TPtr]("id", table, conn),
		events:  db.NewEventStore[E, EPtr]("event_id", eventTable, conn),
		impl:    impl,
	}
}

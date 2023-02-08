// Package models contains tools for working with solve objects stored
// in different databases like SQLite or Postgres.
package models

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/udovin/algo/maps"
	"github.com/udovin/gosql"
	"github.com/udovin/solve/db"
)

type storeIndex[T any] interface {
	Reset()
	Register(object T)
	Deregister(object T)
}

func newIndex[K comparable, T any, TPtr db.ObjectPtr[T]](key func(T) K) *index[K, T, TPtr] {
	return &index[K, T, TPtr]{key: key}
}

type index[K comparable, T any, TPtr db.ObjectPtr[T]] struct {
	key   func(T) K
	index map[K]map[int64]struct{}
}

func (i *index[K, T, TPtr]) Reset() {
	i.index = map[K]map[int64]struct{}{}
}

func (i *index[K, T, TPtr]) Get(key K) map[int64]struct{} {
	return i.index[key]
}

func (i *index[K, T, TPtr]) Register(object T) {
	key := i.key(object)
	id := TPtr(&object).ObjectID()
	if _, ok := i.index[key]; !ok {
		i.index[key] = map[int64]struct{}{}
	}
	i.index[key][id] = struct{}{}
}

func (i *index[K, T, TPtr]) Deregister(object T) {
	key := i.key(object)
	id := TPtr(&object).ObjectID()
	delete(i.index[key], id)
	if len(i.index[key]) == 0 {
		delete(i.index, key)
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

// Cloner represents object that can be cloned.
type Cloner[T any] interface {
	Clone() T
}

type ObjectPtr[T any] interface {
	db.ObjectPtr[T]
	Cloner[T]
}

type ObjectEventPtr[T any, E any] interface {
	db.EventPtr[E]
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

// WithAccountID replaces account ID.
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

type nowKey struct{}

// WithNow replaces time.Now.
func WithNow(ctx context.Context, now time.Time) context.Context {
	return context.WithValue(ctx, nowKey{}, now)
}

// GetNow returns time.Now.
func GetNow(ctx context.Context) time.Time {
	if t, ok := ctx.Value(nowKey{}).(time.Time); ok {
		return t
	}
	return time.Now()
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

type cachedStore[
	T any, E any, TPtr ObjectPtr[T], EPtr ObjectEventPtr[T, E],
] struct {
	db       *gosql.DB
	table    string
	store    db.ObjectStore[T, TPtr]
	events   db.EventStore[E, EPtr]
	consumer db.EventConsumer[E, EPtr]
	impl     baseStoreImpl[T]
	mutex    sync.RWMutex
	objects  *maps.Map[int64, T]
	indexes  []storeIndex[T]
}

// DB returns store database.
func (s *cachedStore[T, E, TPtr, EPtr]) DB() *gosql.DB {
	return s.db
}

func (s *cachedStore[T, E, TPtr, EPtr]) Init(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.initUnlocked(ctx)
}

func (s *cachedStore[T, E, TPtr, EPtr]) initUnlocked(ctx context.Context) error {
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

func (s *cachedStore[T, E, TPtr, EPtr]) initEvents(ctx context.Context) error {
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

func (s *cachedStore[T, E, TPtr, EPtr]) initObjects(ctx context.Context) error {
	rows, err := s.store.LoadObjects(ctx)
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

func (s *cachedStore[T, E, TPtr, EPtr]) Sync(ctx context.Context) error {
	if tx := db.GetTx(ctx); tx != nil {
		return fmt.Errorf("sync cannot be run in transaction")
	}
	return s.consumer.ConsumeEvents(ctx, s.consumeEvent)
}

func (s *cachedStore[T, E, TPtr, EPtr]) newObjectEvent(ctx context.Context, kind EventKind) EPtr {
	var event E
	var eventPtr EPtr = &event
	eventPtr.SetEventTime(time.Now())
	eventPtr.SetEventKind(kind)
	eventPtr.SetEventAccountID(GetAccountID(ctx))
	return eventPtr
}

// Create creates object and returns copy with valid ID.
func (s *cachedStore[T, E, TPtr, EPtr]) Create(ctx context.Context, object TPtr) error {
	eventPtr := s.newObjectEvent(ctx, CreateEvent)
	eventPtr.SetObject(*object)
	if err := s.createObjectEvent(ctx, eventPtr); err != nil {
		return err
	}
	*object = eventPtr.Object()
	return nil
}

// Update updates object with specified ID.
func (s *cachedStore[T, E, TPtr, EPtr]) Update(ctx context.Context, object T) error {
	eventPtr := s.newObjectEvent(ctx, UpdateEvent)
	eventPtr.SetObject(object)
	return s.createObjectEvent(ctx, eventPtr)
}

// Delete deletes compiler with specified ID.
func (s *cachedStore[T, E, TPtr, EPtr]) Delete(ctx context.Context, id int64) error {
	eventPtr := s.newObjectEvent(ctx, DeleteEvent)
	eventPtr.SetObjectID(id)
	return s.createObjectEvent(ctx, eventPtr)
}

// Find finds objects with specified query.
func (s *cachedStore[T, E, TPtr, EPtr]) Find(ctx context.Context, where gosql.BoolExpression) (db.Rows[T], error) {
	return s.store.FindObjects(ctx, where)
}

// Get returns object by id.
//
// Returns sql.ErrNoRows if object does not exist.
func (s *cachedStore[T, E, TPtr, EPtr]) Get(id int64) (T, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if object, ok := s.objects.Get(id); ok {
		return TPtr(&object).Clone(), nil
	}
	var empty T
	return empty, sql.ErrNoRows
}

// All returns all objects contained by this store.
func (s *cachedStore[T, E, TPtr, EPtr]) All() ([]T, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []T
	for it := s.objects.Front(); it != nil; it = it.Next() {
		object := it.Value()
		objects = append(objects, TPtr(&object).Clone())
	}
	return objects, nil
}

var (
	sqlRepeatableRead = gosql.WithIsolation(sql.LevelRepeatableRead)
	sqlReadOnly       = gosql.WithReadOnly(true)
)

func (s *cachedStore[T, E, TPtr, EPtr]) createObjectEvent(
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
		if err := s.store.CreateObject(ctx, &object); err != nil {
			return err
		}
		eventPtr.SetObject(object)
	case UpdateEvent:
		if err := s.store.UpdateObject(ctx, &object); err != nil {
			return err
		}
		eventPtr.SetObject(object)
	case DeleteEvent:
		if err := s.store.DeleteObject(ctx, eventPtr.ObjectID()); err != nil {
			return err
		}
	}
	return s.events.CreateEvent(ctx, eventPtr)
}

func (s *cachedStore[T, E, TPtr, EPtr]) lockStore(tx *sql.Tx) error {
	switch s.db.Dialect() {
	case gosql.SQLiteDialect:
		return nil
	default:
		_, err := tx.Exec(fmt.Sprintf("LOCK TABLE %q", s.table))
		return err
	}
}

//lint:ignore U1000 Used in generic interface.
func (s *cachedStore[T, E, TPtr, EPtr]) reset() {
	for _, index := range s.indexes {
		index.Reset()
	}
	s.objects = maps.NewMap[int64, T](lessInt64)
}

func lessInt64(lhs, rhs int64) bool {
	return lhs < rhs
}

//lint:ignore U1000 Used in generic interface.
func (s *cachedStore[T, E, TPtr, EPtr]) onCreateObject(object T) {
	id := TPtr(&object).ObjectID()
	s.objects.Set(id, object)
	for _, index := range s.indexes {
		index.Register(object)
	}
}

//lint:ignore U1000 Used in generic interface.
func (s *cachedStore[T, E, TPtr, EPtr]) onDeleteObject(id int64) {
	if it := s.objects.Find(id); it != nil {
		for _, index := range s.indexes {
			index.Deregister(it.Value())
		}
		s.objects.Erase(it)
	}
}

//lint:ignore U1000 Used in generic interface.
func (s *cachedStore[T, E, TPtr, EPtr]) onUpdateObject(object T) {
	s.impl.onDeleteObject(TPtr(&object).ObjectID())
	s.impl.onCreateObject(object)
}

func (s *cachedStore[T, E, TPtr, EPtr]) consumeEvent(event E) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
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

func makeBaseStore[T any, E any, TPtr ObjectPtr[T], EPtr ObjectEventPtr[T, E]](
	conn *gosql.DB,
	table, eventTable string,
	impl baseStoreImpl[T],
	indexes ...storeIndex[T],
) cachedStore[T, E, TPtr, EPtr] {
	return cachedStore[T, E, TPtr, EPtr]{
		db:      conn,
		table:   table,
		store:   db.NewObjectStore[T, TPtr]("id", table, conn),
		events:  db.NewEventStore[E, EPtr]("event_id", eventTable, conn),
		impl:    impl,
		indexes: indexes,
	}
}

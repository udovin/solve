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

// CachedStore represents cached store.
type CachedStore interface {
	Init(ctx context.Context) error
	Sync(ctx context.Context) error
}

type cachedStoreImpl[T any] interface {
	reset()
	onCreateObject(T)
	onDeleteObject(int64)
	onUpdateObject(T)
}

type cachedStore[
	T any, E any, TPtr ObjectPtr[T], EPtr ObjectEventPtr[T, E],
] struct {
	db       *gosql.DB
	table    string
	store    db.ObjectStore[T, TPtr]
	events   db.EventStore[E, EPtr]
	consumer db.EventConsumer[E, EPtr]
	impl     cachedStoreImpl[T]
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
func (s *cachedStore[T, E, TPtr, EPtr]) Get(_ context.Context, id int64) (T, error) {
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

func makeCachedStore[T any, E any, TPtr ObjectPtr[T], EPtr ObjectEventPtr[T, E]](
	conn *gosql.DB,
	table, eventTable string,
	impl cachedStoreImpl[T],
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

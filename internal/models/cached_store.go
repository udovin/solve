package models

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/udovin/algo/btree"
	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
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
	objects  btree.Map[int64, T]
	indexes  []storeIndex[T]
	syncTime time.Time
}

// DB returns store database.
func (s *cachedStore[T, E, TPtr, EPtr]) DB() *gosql.DB {
	return s.db
}

func (s *cachedStore[T, E, TPtr, EPtr]) Objects() db.ObjectROStore[T] {
	return s.store
}

func (s *cachedStore[T, E, TPtr, EPtr]) Events() db.EventROStore[E] {
	return s.events
}

func (s *cachedStore[T, E, TPtr, EPtr]) Init(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	t := time.Now()
	if err := s.initUnlocked(ctx); err != nil {
		return err
	}
	s.syncTime = t
	return nil
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
	t := time.Now()
	if err := s.consumer.ConsumeEvents(ctx, s.consumeEvent); err != nil {
		return err
	}
	s.updateSync(t)
	return nil
}

func (s *cachedStore[T, E, TPtr, EPtr]) updateSync(t time.Time) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.syncTime = t
}

func (s *cachedStore[T, E, TPtr, EPtr]) needSync(ctx context.Context) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	t, ok := ctx.Value(syncKey{}).(time.Time)
	return ok && !s.syncTime.After(t)
}

// TrySync updates store cache only when needed.
func (s *cachedStore[T, E, TPtr, EPtr]) TrySync(ctx context.Context) error {
	if !s.needSync(ctx) {
		return nil
	}
	return s.Sync(ctx)
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
func (s *cachedStore[T, E, TPtr, EPtr]) Find(
	ctx context.Context, options ...db.FindObjectsOption,
) (db.Rows[T], error) {
	return s.store.FindObjects(ctx, options...)
}

// FindOne finds one object with specified query.
func (s *cachedStore[T, E, TPtr, EPtr]) FindOne(
	ctx context.Context, options ...db.FindObjectsOption,
) (T, error) {
	var empty T
	rows, err := s.Find(ctx, append(options, db.WithLimit(1))...)
	if err != nil {
		return empty, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return empty, err
		}
		return empty, sql.ErrNoRows
	}
	return rows.Row(), nil
}

type syncKey struct{}

func WithSync(ctx context.Context) context.Context {
	return context.WithValue(ctx, syncKey{}, time.Now())
}

// Get returns object by id.
//
// Returns sql.ErrNoRows if object does not exist.
func (s *cachedStore[T, E, TPtr, EPtr]) Get(ctx context.Context, id int64) (T, error) {
	var empty T
	if err := s.TrySync(ctx); err != nil {
		return empty, err
	}
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if object, ok := s.objects.Get(id); ok {
		return TPtr(&object).Clone(), nil
	}
	return empty, sql.ErrNoRows
}

type btreeRows[T any, TPtr ObjectPtr[T]] struct {
	iter    btree.MapIter[int64, T]
	mutex   sync.Locker
	limit   int
	beginID int64
}

func (r *btreeRows[T, TPtr]) Next() bool {
	if r.limit <= 0 {
		r.Close()
		return false
	}
	r.limit--
	if r.beginID > 0 {
		hasNext := r.iter.Seek(r.beginID)
		r.beginID = 0
		return hasNext
	}
	return r.iter.Next()
}

func (r *btreeRows[T, TPtr]) Row() T {
	value := r.iter.Value()
	return TPtr(&value).Clone()
}

func (r *btreeRows[T, TPtr]) Err() error {
	return nil
}

func (r *btreeRows[T, TPtr]) Close() error {
	if r.mutex == nil {
		return nil
	}
	r.mutex.Unlock()
	r.mutex = nil
	return nil
}

type btreeReverseRows[T any, TPtr ObjectPtr[T]] struct {
	iter    btree.MapIter[int64, T]
	mutex   sync.Locker
	limit   int
	beginID int64
}

func (r *btreeReverseRows[T, TPtr]) Next() bool {
	if r.limit <= 0 {
		r.Close()
		return false
	}
	r.limit--
	if r.beginID > 0 {
		hasNext := r.iter.Seek(r.beginID + 1)
		r.beginID = 0
		if !hasNext {
			return r.iter.Last()
		}
	}
	return r.iter.Prev()
}

func (r *btreeReverseRows[T, TPtr]) Row() T {
	value := r.iter.Value()
	return TPtr(&value).Clone()
}

func (r *btreeReverseRows[T, TPtr]) Err() error {
	return nil
}

func (r *btreeReverseRows[T, TPtr]) Close() error {
	if r.mutex == nil {
		return nil
	}
	r.mutex.Unlock()
	r.mutex = nil
	return nil
}

// All returns all objects contained by this store.
func (s *cachedStore[T, E, TPtr, EPtr]) All(ctx context.Context, limit int, beginID int64) (db.Rows[T], error) {
	if limit <= 0 {
		limit = math.MaxInt
	}
	s.mutex.RLock()
	return &btreeRows[T, TPtr]{
		iter:    s.objects.Iter(),
		mutex:   s.mutex.RLocker(),
		limit:   limit,
		beginID: beginID,
	}, nil
}

// ReverseAll returns all objects contained by this store.
func (s *cachedStore[T, E, TPtr, EPtr]) ReverseAll(ctx context.Context, limit int, beginID int64) (db.Rows[T], error) {
	if limit <= 0 {
		limit = math.MaxInt
	}
	s.mutex.RLock()
	return &btreeReverseRows[T, TPtr]{
		iter:    s.objects.Iter(),
		mutex:   s.mutex.RLocker(),
		limit:   limit,
		beginID: beginID,
	}, nil
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
	s.objects = btree.NewMap[int64, T](lessInt64)
}

func lessInt64(lhs, rhs int64) bool {
	return lhs < rhs
}

func lessString(lhs, rhs string) bool {
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
	if object, ok := s.objects.Get(id); ok {
		for _, index := range s.indexes {
			index.Deregister(object)
		}
		s.objects.Delete(id)
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

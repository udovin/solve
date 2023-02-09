// Package models contains tools for working with solve objects stored
// in different databases like SQLite or Postgres.
package models

import (
	"context"
	"fmt"
	"time"

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

type baseStore[
	T any, E any, TPtr ObjectPtr[T], EPtr ObjectEventPtr[T, E],
] struct {
	db *gosql.DB
}

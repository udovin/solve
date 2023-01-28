package models

import (
	"github.com/udovin/gosql"
)

// InternalGroup represents an internal group.
type InternalGroup struct {
	baseObject
	OwnerID NInt64 `db:"owner_id"`
	Title   string `db:"title"`
}

// Clone creates copy of internal group.
func (o InternalGroup) Clone() InternalGroup {
	return o
}

// AccountEvent represents an internal group event.
type InternalGroupEvent struct {
	baseEvent
	InternalGroup
}

// Object returns event internal group.
func (e InternalGroupEvent) Object() InternalGroup {
	return e.InternalGroup
}

// SetObject sets event account.
func (e *InternalGroupEvent) SetObject(o InternalGroup) {
	e.InternalGroup = o
}

// InternalGroupStore represents store for internal groups.
type InternalGroupStore struct {
	baseStore[InternalGroup, InternalGroupEvent, *InternalGroup, *InternalGroupEvent]
}

var _ baseStoreImpl[InternalGroup] = (*InternalGroupStore)(nil)

// NewInternalGroupStore creates a new instance of InternalGroupStore.
func NewInternalGroupStore(
	db *gosql.DB, table, eventTable string,
) *InternalGroupStore {
	impl := &InternalGroupStore{}
	impl.baseStore = makeBaseStore[InternalGroup, InternalGroupEvent](
		db, table, eventTable, impl,
	)
	return impl
}

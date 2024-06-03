package models

import (
	"github.com/udovin/gosql"
)

// Group represents a group for users.
type Group struct {
	baseObject
	OwnerID NInt64 `db:"owner_id"`
	Title   string `db:"title"`
}

// AccountKind returns GroupAccount kind.
func (o Group) AccountKind() AccountKind {
	return GroupAccountKind
}

// Clone creates copy of group.
func (o Group) Clone() Group {
	return o
}

// GroupEvent represents an group event.
type GroupEvent struct {
	baseEvent
	Group
}

// Object returns event group.
func (e GroupEvent) Object() Group {
	return e.Group
}

// SetObject sets event group.
func (e *GroupEvent) SetObject(o Group) {
	e.Group = o
}

// GroupStore represents store for groups.
type GroupStore struct {
	cachedStore[Group, GroupEvent, *Group, *GroupEvent]
}

// NewGroupStore creates a new instance of GroupStore.
func NewGroupStore(
	db *gosql.DB, table, eventTable string,
) *GroupStore {
	impl := &GroupStore{}
	impl.cachedStore = makeCachedManualStore[Group, GroupEvent](
		db, table, eventTable, impl,
	)
	return impl
}

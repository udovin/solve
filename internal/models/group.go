package models

import (
	"context"

	"github.com/udovin/gosql"
)

// Group represents a group for users.
type Group struct {
	baseObject
	AccountID int64  `db:"account_id"`
	OwnerID   NInt64 `db:"owner_id"`
	Title     string `db:"title"`
}

// AccountKind returns GroupAccount kind.
func (o Group) AccountKind() AccountKind {
	return GroupAccount
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
	byAccount *btreeIndex[int64, Group, *Group]
}

// GetByAccount returns group user by account id.
func (s *GroupStore) GetByAccount(ctx context.Context, accountID int64) (Group, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return btreeIndexGet(s.byAccount, s.objects.Iter(), accountID)
}

// NewGroupStore creates a new instance of GroupStore.
func NewGroupStore(
	db *gosql.DB, table, eventTable string,
) *GroupStore {
	impl := &GroupStore{
		byAccount: newBTreeIndex(
			func(o Group) (int64, bool) { return o.AccountID, true },
			lessInt64,
		),
	}
	impl.cachedStore = makeCachedStore[Group, GroupEvent](
		db, table, eventTable, impl, impl.byAccount,
	)
	return impl
}

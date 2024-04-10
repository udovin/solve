package models

import (
	"context"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
)

type MemberKind int64

const (
	RegularMember MemberKind = 1
	ManagerMember MemberKind = 2
)

// Group represents a group member for users.
type GroupMember struct {
	baseObject
	GroupID   int64      `db:"group_id"`
	AccountID int64      `db:"account_id"`
	Kind      MemberKind `db:"kind"`
}

// Clone creates copy of group member.
func (o GroupMember) Clone() GroupMember {
	return o
}

// GroupMemberEvent represents an group member event.
type GroupMemberEvent struct {
	baseEvent
	GroupMember
}

// Object returns event group member.
func (e GroupMemberEvent) Object() GroupMember {
	return e.GroupMember
}

// SetObject sets event group member.
func (e *GroupMemberEvent) SetObject(o GroupMember) {
	e.GroupMember = o
}

// GroupMemberStore represents store for group members.
type GroupMemberStore struct {
	cachedStore[GroupMember, GroupMemberEvent, *GroupMember, *GroupMemberEvent]
	byGroup   *btreeIndex[int64, GroupMember, *GroupMember]
	byAccount *btreeIndex[int64, GroupMember, *GroupMember]
}

// GetByGroup returns group members by group id.
func (s *GroupMemberStore) FindByGroup(ctx context.Context, groupID ...int64) (db.Rows[GroupMember], error) {
	s.mutex.RLock()
	return btreeIndexFind(
		s.byGroup,
		s.objects.Iter(),
		s.mutex.RLocker(),
		groupID,
		0,
	), nil
}

// GetByAccount returns group members by account id.
func (s *GroupMemberStore) FindByAccount(ctx context.Context, accountID ...int64) (db.Rows[GroupMember], error) {
	s.mutex.RLock()
	return btreeIndexFind(
		s.byAccount,
		s.objects.Iter(),
		s.mutex.RLocker(),
		accountID,
		0,
	), nil
}

// NewGroupMemberStore creates a new instance of GroupMemberStore.
func NewGroupMemberStore(
	db *gosql.DB, table, eventTable string,
) *GroupMemberStore {
	impl := &GroupMemberStore{
		byGroup:   newBTreeIndex(func(o GroupMember) (int64, bool) { return o.GroupID, true }, lessInt64),
		byAccount: newBTreeIndex(func(o GroupMember) (int64, bool) { return o.AccountID, true }, lessInt64),
	}
	impl.cachedStore = makeCachedStore[GroupMember, GroupMemberEvent](
		db, table, eventTable, impl, impl.byGroup, impl.byAccount,
	)
	return impl
}

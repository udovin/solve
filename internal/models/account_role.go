package models

import (
	"context"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
)

// AccountRole represents a account role.
type AccountRole struct {
	baseObject
	// AccountID contains account ID.
	AccountID int64 `db:"account_id"`
	// RoleID contains role ID.
	RoleID int64 `db:"role_id"`
}

// Clone creates copy of account role.
func (o AccountRole) Clone() AccountRole {
	return o
}

// AccountRoleEvent represents account role event.
type AccountRoleEvent struct {
	baseEvent
	AccountRole
}

// Object returns account role.
func (e AccountRoleEvent) Object() AccountRole {
	return e.AccountRole
}

// SetObject sets event account role.
func (e *AccountRoleEvent) SetObject(o AccountRole) {
	e.AccountRole = o
}

// AccountRoleStore represents store for account roles.
type AccountRoleStore struct {
	cachedStore[AccountRole, AccountRoleEvent, *AccountRole, *AccountRoleEvent]
	byAccount *btreeIndex[int64, AccountRole, *AccountRole]
}

// FindByAccount returns roles by account ID.
func (s *AccountRoleStore) FindByAccount(ctx context.Context, accountID ...int64) (db.Rows[AccountRole], error) {
	s.mutex.RLock()
	return btreeIndexFind(
		s.byAccount,
		s.objects.Iter(),
		s.mutex.RLocker(),
		accountID,
		0,
	), nil
}

// NewAccountRoleStore creates a new instance of AccountRoleStore.
func NewAccountRoleStore(
	db *gosql.DB, table, eventTable string,
) *AccountRoleStore {
	impl := &AccountRoleStore{
		byAccount: newBTreeIndex(func(o AccountRole) (int64, bool) { return o.AccountID, true }, lessInt64),
	}
	impl.cachedStore = makeCachedStore[AccountRole, AccountRoleEvent](
		db, table, eventTable, impl, impl.byAccount,
	)
	return impl
}

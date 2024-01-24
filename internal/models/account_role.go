package models

import (
	"github.com/udovin/gosql"
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
	byAccount *index[int64, AccountRole, *AccountRole]
}

// FindByAccount returns roles by account ID.
func (s *AccountRoleStore) FindByAccount(id int64) ([]AccountRole, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []AccountRole
	for id := range s.byAccount.Get(id) {
		if object, ok := s.objects.Get(id); ok {
			objects = append(objects, object.Clone())
		}
	}
	return objects, nil
}

// NewAccountRoleStore creates a new instance of AccountRoleStore.
func NewAccountRoleStore(
	db *gosql.DB, table, eventTable string,
) *AccountRoleStore {
	impl := &AccountRoleStore{
		byAccount: newIndex(func(o AccountRole) (int64, bool) { return o.AccountID, true }),
	}
	impl.cachedStore = makeCachedStore[AccountRole, AccountRoleEvent](
		db, table, eventTable, impl, impl.byAccount,
	)
	return impl
}

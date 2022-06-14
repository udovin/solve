package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

// AccountRole represents a account role.
type AccountRole struct {
	// ID contains ID of account role.
	ID int64 `db:"id"`
	// AccountID contains account ID.
	AccountID int64 `db:"account_id"`
	// RoleID contains role ID.
	RoleID int64 `db:"role_id"`
}

// ObjectID return ID of account role.
func (o AccountRole) ObjectID() int64 {
	return o.ID
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
	baseStore[AccountRole, AccountRoleEvent]
	roles     map[int64]AccountRole
	byAccount index[int64]
}

// Get returns account role by ID.
//
// If there is no role with specified id then
// sql.ErrNoRows will be returned.
func (s *AccountRoleStore) Get(id int64) (AccountRole, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if role, ok := s.roles[id]; ok {
		return role.Clone(), nil
	}
	return AccountRole{}, sql.ErrNoRows
}

// FindByAccount returns roles by account ID.
func (s *AccountRoleStore) FindByAccount(id int64) ([]AccountRole, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var roles []AccountRole
	for id := range s.byAccount[id] {
		if role, ok := s.roles[id]; ok {
			roles = append(roles, role.Clone())
		}
	}
	return roles, nil
}

func (s *AccountRoleStore) reset() {
	s.roles = map[int64]AccountRole{}
	s.byAccount = makeIndex[int64]()
}

func (s *AccountRoleStore) makeObject(id int64) AccountRole {
	return AccountRole{ID: id}
}

func (s *AccountRoleStore) makeObjectEvent(typ EventType) AccountRoleEvent {
	return AccountRoleEvent{baseEvent: makeBaseEvent(typ)}
}

func (s *AccountRoleStore) onCreateObject(role AccountRole) {
	s.roles[role.ID] = role
	s.byAccount.Create(role.AccountID, role.ID)
}

func (s *AccountRoleStore) onDeleteObject(id int64) {
	if role, ok := s.roles[id]; ok {
		s.byAccount.Delete(role.AccountID, role.ID)
		delete(s.roles, role.ID)
	}
}

var _ baseStoreImpl[AccountRole, AccountRoleEvent] = (*AccountRoleStore)(nil)

// NewAccountRoleStore creates a new instance of AccountRoleStore.
func NewAccountRoleStore(
	db *gosql.DB, table, eventTable string,
) *AccountRoleStore {
	impl := &AccountRoleStore{}
	impl.baseStore = makeBaseStore[AccountRole, AccountRoleEvent](
		db, table, eventTable, impl,
	)
	return impl
}

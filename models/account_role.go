package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
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
func (e AccountRoleEvent) Object() db.Object {
	return e.AccountRole
}

// WithObject return event with replaced account role.
func (e AccountRoleEvent) WithObject(o db.Object) ObjectEvent {
	e.AccountRole = o.(AccountRole)
	return e
}

// AccountRoleStore represents store for account roles.
type AccountRoleStore struct {
	baseStore
	roles     map[int64]AccountRole
	byAccount indexInt64
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

// CreateTx creates account role and returns copy with valid ID.
func (s *AccountRoleStore) CreateTx(
	tx *sql.Tx, role AccountRole,
) (AccountRole, error) {
	event, err := s.createObjectEvent(tx, AccountRoleEvent{
		makeBaseEvent(CreateEvent),
		role,
	})
	if err != nil {
		return AccountRole{}, err
	}
	return event.Object().(AccountRole), nil
}

// UpdateTx updates account role with specified ID.
func (s *AccountRoleStore) UpdateTx(tx *sql.Tx, role AccountRole) error {
	_, err := s.createObjectEvent(tx, AccountRoleEvent{
		makeBaseEvent(UpdateEvent),
		role,
	})
	return err
}

// DeleteTx deletes account role with specified ID.
func (s *AccountRoleStore) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := s.createObjectEvent(tx, AccountRoleEvent{
		makeBaseEvent(DeleteEvent),
		AccountRole{ID: id},
	})
	return err
}

func (s *AccountRoleStore) reset() {
	s.roles = map[int64]AccountRole{}
	s.byAccount = indexInt64{}
}

func (s *AccountRoleStore) onCreateObject(o db.Object) {
	role := o.(AccountRole)
	s.roles[role.ID] = role
	s.byAccount.Create(role.AccountID, role.ID)
}

func (s *AccountRoleStore) onDeleteObject(o db.Object) {
	role := o.(AccountRole)
	s.byAccount.Delete(role.AccountID, role.ID)
	delete(s.roles, role.ID)
}

func (s *AccountRoleStore) onUpdateObject(o db.Object) {
	role := o.(AccountRole)
	if old, ok := s.roles[role.ID]; ok {
		if old.AccountID != role.AccountID {
			s.byAccount.Delete(old.AccountID, old.ID)
		}
	}
	s.onCreateObject(o)
}

// NewAccountRoleStore creates a new instance of AccountRoleStore.
func NewAccountRoleStore(
	table, eventTable string, dialect db.Dialect,
) *AccountRoleStore {
	impl := &AccountRoleStore{}
	impl.baseStore = makeBaseStore(
		AccountRole{}, table, AccountRoleEvent{}, eventTable, impl, dialect,
	)
	return impl
}

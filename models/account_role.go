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

func (o AccountRole) clone() AccountRole {
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

// AccountRoleManager represents manager for account roles.
type AccountRoleManager struct {
	baseManager
	roles     map[int64]AccountRole
	byAccount indexInt64
}

// Get returns account role by ID.
//
// If there is no role with specified id then
// sql.ErrNoRows will be returned.
func (m *AccountRoleManager) Get(id int64) (AccountRole, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if role, ok := m.roles[id]; ok {
		return role.clone(), nil
	}
	return AccountRole{}, sql.ErrNoRows
}

// FindByAccount returns roles by account ID.
func (m *AccountRoleManager) FindByAccount(id int64) ([]AccountRole, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var roles []AccountRole
	for id := range m.byAccount[id] {
		if role, ok := m.roles[id]; ok {
			roles = append(roles, role.clone())
		}
	}
	return roles, nil
}

// CreateTx creates account role and returns copy with valid ID.
func (m *AccountRoleManager) CreateTx(
	tx *sql.Tx, role AccountRole,
) (AccountRole, error) {
	event, err := m.createObjectEvent(tx, AccountRoleEvent{
		makeBaseEvent(CreateEvent),
		role,
	})
	if err != nil {
		return AccountRole{}, err
	}
	return event.Object().(AccountRole), nil
}

// UpdateTx updates account role with specified ID.
func (m *AccountRoleManager) UpdateTx(tx *sql.Tx, role AccountRole) error {
	_, err := m.createObjectEvent(tx, AccountRoleEvent{
		makeBaseEvent(UpdateEvent),
		role,
	})
	return err
}

// DeleteTx deletes account role with specified ID.
func (m *AccountRoleManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, AccountRoleEvent{
		makeBaseEvent(DeleteEvent),
		AccountRole{ID: id},
	})
	return err
}

func (m *AccountRoleManager) reset() {
	m.roles = map[int64]AccountRole{}
	m.byAccount = indexInt64{}
}

func (m *AccountRoleManager) onCreateObject(o db.Object) {
	role := o.(AccountRole)
	m.roles[role.ID] = role
	m.byAccount.Create(role.AccountID, role.ID)
}

func (m *AccountRoleManager) onDeleteObject(o db.Object) {
	role := o.(AccountRole)
	m.byAccount.Delete(role.AccountID, role.ID)
	delete(m.roles, role.ID)
}

func (m *AccountRoleManager) onUpdateObject(o db.Object) {
	role := o.(AccountRole)
	if old, ok := m.roles[role.ID]; ok {
		if old.AccountID != role.AccountID {
			m.byAccount.Delete(old.AccountID, old.ID)
		}
	}
	m.onCreateObject(o)
}

// NewAccountRoleManager creates a new instance of AccountRoleManager.
func NewAccountRoleManager(
	table, eventTable string, dialect db.Dialect,
) *AccountRoleManager {
	impl := &AccountRoleManager{}
	impl.baseManager = makeBaseManager(
		AccountRole{}, table, AccountRoleEvent{}, eventTable, impl, dialect,
	)
	return impl
}

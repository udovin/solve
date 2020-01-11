package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
)

// UserRole represents a user role.
type UserRole struct {
	// ID contains ID of user role.
	ID int64 `db:"id" json:""`
	// RoleID contains role ID.
	RoleID int64 `db:"role_id" json:""`
	// UserID contains user ID.
	UserID int64 `db:"user_id" json:""`
}

// ObjectID return ID of user role.
func (o UserRole) ObjectID() int64 {
	return o.ID
}

// UserRoleEvent represents user role event.
type UserRoleEvent struct {
	baseEvent
	UserRole
}

// Object returns user role.
func (e UserRoleEvent) Object() db.Object {
	return e.UserRole
}

// WithObject return event with replaced user role.
func (e UserRoleEvent) WithObject(o db.Object) ObjectEvent {
	e.UserRole = o.(UserRole)
	return e
}

// UserRoleManager represents manager for user roles.
type UserRoleManager struct {
	baseManager
	roles  map[int64]UserRole
	byUser indexInt64
}

// Get returns user role by ID.
//
// If there is no role with specified id then
// sql.ErrNoRows will be returned.
func (m *UserRoleManager) Get(id int64) (UserRole, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if role, ok := m.roles[id]; ok {
		return role, nil
	}
	return UserRole{}, sql.ErrNoRows
}

// FindByUser returns roles by user ID.
func (m *UserRoleManager) FindByUser(id int64) ([]UserRole, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var roles []UserRole
	for id := range m.byUser[id] {
		if role, ok := m.roles[id]; ok {
			roles = append(roles, role)
		}
	}
	return roles, nil
}

// CreateTx creates user role and returns copy with valid ID.
func (m *UserRoleManager) CreateTx(
	tx *sql.Tx, role UserRole,
) (UserRole, error) {
	event, err := m.createObjectEvent(tx, UserRoleEvent{
		makeBaseEvent(CreateEvent),
		role,
	})
	if err != nil {
		return UserRole{}, err
	}
	return event.Object().(UserRole), nil
}

// UpdateTx updates user role with specified ID.
func (m *UserRoleManager) UpdateTx(tx *sql.Tx, role UserRole) error {
	_, err := m.createObjectEvent(tx, UserRoleEvent{
		makeBaseEvent(UpdateEvent),
		role,
	})
	return err
}

// DeleteTx deletes user role with specified ID.
func (m *UserRoleManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, UserRoleEvent{
		makeBaseEvent(DeleteEvent),
		UserRole{ID: id},
	})
	return err
}

func (m *UserRoleManager) reset() {
	m.roles = map[int64]UserRole{}
	m.byUser = indexInt64{}
}

func (m *UserRoleManager) onCreateObject(o db.Object) {
	role := o.(UserRole)
	m.roles[role.ID] = role
	m.byUser.Create(role.UserID, role.ID)
}

func (m *UserRoleManager) onDeleteObject(o db.Object) {
	role := o.(UserRole)
	m.byUser.Delete(role.UserID, role.ID)
	delete(m.roles, role.ID)
}

func (m *UserRoleManager) onUpdateObject(o db.Object) {
	role := o.(UserRole)
	if old, ok := m.roles[role.ID]; ok {
		if old.UserID != role.UserID {
			m.byUser.Delete(old.UserID, old.ID)
		}
	}
	m.onCreateObject(o)
}

// NewUserRoleManager creates a new instance of UserRoleManager.
func NewUserRoleManager(
	table, eventTable string, dialect db.Dialect,
) *UserRoleManager {
	impl := &UserRoleManager{}
	impl.baseManager = makeBaseManager(
		UserRole{}, table, UserRoleEvent{}, eventTable, impl, dialect,
	)
	return impl
}

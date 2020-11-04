package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
)

// Role represents a role.
type Role struct {
	// ID contains ID of role.
	ID int64 `db:"id" json:"id"`
	// Code contains role code.
	//
	// Code should be unique for all roles in the store.
	Code string `db:"code" json:"code"`
}

const (
	// LoginRole represents name of role for login action.
	LoginRole = "login"
	// LogoutRole represents name of role for logout action.
	LogoutRole = "logout"
	// RegisterRole represents name of role for register action.
	RegisterRole = "register"
	// AuthStatusRole represents name of role for auth status check.
	AuthStatusRole = "auth_status"
	// ObserveRoleRole represents name of role for observing role.
	ObserveRoleRole = "observe_role"
	// CreateRoleRole represents name of role for creating new role.
	CreateRoleRole = "create_role"
	// DeleteRoleRole represents name of role for deleting new role.
	DeleteRoleRole = "delete_role"
	// ObserveUserRoleRole represents name of role for observing user role.
	ObserveUserRoleRole = "observe_user_role"
	// CreateUserRoleRole represents name of role for attaching role to user.
	CreateUserRoleRole = "create_user_role"
	// DeleteUserRoleRole represents name of role for detaching role from user.
	DeleteUserRoleRole = "delete_user_role"
	// GuestGroupRole represents name of role for guest group.
	GuestGroupRole = "guest_group"
	// UserGroupRole represents name of role for user group.
	UserGroupRole = "user_group"
)

var builtInRoles = map[string]struct{}{
	LoginRole:           {},
	LogoutRole:          {},
	RegisterRole:        {},
	AuthStatusRole:      {},
	ObserveRoleRole:     {},
	CreateRoleRole:      {},
	DeleteRoleRole:      {},
	ObserveUserRoleRole: {},
	CreateUserRoleRole:  {},
	DeleteUserRoleRole:  {},
	GuestGroupRole:      {},
	UserGroupRole:       {},
}

// ObjectID return ID of role.
func (o Role) ObjectID() int64 {
	return o.ID
}

// IsBuiltIn returns flag that role is built-in.
func (o Role) IsBuiltIn() bool {
	_, ok := builtInRoles[o.Code]
	return ok
}

func (o Role) clone() Role {
	return o
}

// RoleEvent represents role event.
type RoleEvent struct {
	baseEvent
	Role
}

// Object returns event role.
func (e RoleEvent) Object() db.Object {
	return e.Role
}

// WithObject returns event with replaced Role.
func (e RoleEvent) WithObject(o db.Object) ObjectEvent {
	e.Role = o.(Role)
	return e
}

// RoleManager represents a role manager.
type RoleManager struct {
	baseManager
	roles  map[int64]Role
	byCode map[string]int64
}

// Get returns role by ID.
//
// If there is no role with specified ID then
// sql.ErrNoRows will be returned.
func (m *RoleManager) Get(id int64) (Role, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if role, ok := m.roles[id]; ok {
		return role.clone(), nil
	}
	return Role{}, sql.ErrNoRows
}

// GetByCode returns role by code.
//
// If there is no role with specified code then
// sql.ErrNoRows will be returned.
func (m *RoleManager) GetByCode(code string) (Role, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if id, ok := m.byCode[code]; ok {
		if role, ok := m.roles[id]; ok {
			return role.clone(), nil
		}
	}
	return Role{}, sql.ErrNoRows
}

// CreateTx creates role and returns copy with valid ID.
func (m *RoleManager) CreateTx(tx *sql.Tx, role Role) (Role, error) {
	event, err := m.createObjectEvent(tx, RoleEvent{
		makeBaseEvent(CreateEvent),
		role,
	})
	if err != nil {
		return Role{}, err
	}
	return event.Object().(Role), nil
}

// UpdateTx updates role with specified ID.
func (m *RoleManager) UpdateTx(tx *sql.Tx, role Role) error {
	_, err := m.createObjectEvent(tx, RoleEvent{
		makeBaseEvent(UpdateEvent),
		role,
	})
	return err
}

// DeleteTx deletes role with specified ID.
func (m *RoleManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, RoleEvent{
		makeBaseEvent(DeleteEvent),
		Role{ID: id},
	})
	return err
}

func (m *RoleManager) reset() {
	m.roles = map[int64]Role{}
	m.byCode = map[string]int64{}
}

func (m *RoleManager) onCreateObject(o db.Object) {
	role := o.(Role)
	m.roles[role.ID] = role
	m.byCode[role.Code] = role.ID
}

func (m *RoleManager) onDeleteObject(o db.Object) {
	role := o.(Role)
	delete(m.byCode, role.Code)
	delete(m.roles, role.ID)
}

func (m *RoleManager) onUpdateObject(o db.Object) {
	role := o.(Role)
	if old, ok := m.roles[role.ID]; ok {
		if old.Code != role.Code {
			delete(m.byCode, old.Code)
		}
	}
	m.onCreateObject(o)
}

// NewRoleManager creates a new instance of RoleManager.
func NewRoleManager(
	table, eventTable string, dialect db.Dialect,
) *RoleManager {
	impl := &RoleManager{}
	impl.baseManager = makeBaseManager(
		Role{}, table, RoleEvent{}, eventTable, impl, dialect,
	)
	return impl
}

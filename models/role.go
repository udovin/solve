package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
)

// Role represents a role.
type Role struct {
	// ID contains ID of role.
	ID int64 `db:"id" json:""`
	// ParentID contains ID of parent role.
	ParentID NInt64 `db:"parent_id" json:",omitempty"`
	// Code contains role code.
	//
	// Code should be unique for all roles in the store.
	Code string `db:"code" json:""`
}

const (
	LoginRole      = "Login"
	LogoutRole     = "Logout"
	RegisterRole   = "Register"
	AuthStatusRole = "AuthStatus"
)

// ObjectID return ID of role.
func (o Role) ObjectID() int64 {
	return o.ID
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
		return role, nil
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
			return role, nil
		}
	}
	return Role{}, sql.ErrNoRows
}

func (m *RoleManager) reset() {
	m.roles = map[int64]Role{}
	m.byCode = map[string]int64{}
}

func (m *RoleManager) addObject(o db.Object) {
	m.onCreateObject(o)
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

func NewRoleManager(
	table, eventTable string, dialect db.Dialect,
) *RoleManager {
	impl := &RoleManager{}
	impl.baseManager = makeBaseManager(
		Role{}, table, RoleEvent{}, eventTable, impl, dialect,
	)
	return impl
}

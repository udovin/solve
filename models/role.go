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
	// Code should be unique for all roles in the events.
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
	// ObserveUserRole represents name of role for observing user.
	ObserveUserRole = "observe_user"
	// ObserveUserEmailRole represents name of role for observing user email.
	ObserveUserEmailRole = "observe_user_email"
	// ObserveUserFirstNameRole represents name of role for observing
	// user first name.
	ObserveUserFirstNameRole = "observe_user_first_name"
	// ObserveUserLastNameRole represents name of role for observing
	// user last name.
	ObserveUserLastNameRole = "observe_user_last_name"
	// ObserveUserMiddleNameRole represents name of role for observing
	// user middle name.
	ObserveUserMiddleNameRole = "observe_user_middle_name"
	// ObserveUserSessionRole represents name of role for observing
	// user sessions.
	ObserveUserSessionRole = "observe_user_session"
	// ObserveSessionRole represents role for observing session.
	ObserveSessionRole = "observe_session"
	// DeleteSessionRole represents role for deleting session.
	DeleteSessionRole = "delete_session"
	// GuestGroupRole represents name of role for guest group.
	GuestGroupRole = "guest_group"
	// UserGroupRole represents name of role for user group.
	UserGroupRole = "user_group"
)

var builtInRoles = map[string]struct{}{
	LoginRole:                 {},
	LogoutRole:                {},
	RegisterRole:              {},
	AuthStatusRole:            {},
	ObserveRoleRole:           {},
	CreateRoleRole:            {},
	DeleteRoleRole:            {},
	ObserveUserRoleRole:       {},
	CreateUserRoleRole:        {},
	DeleteUserRoleRole:        {},
	ObserveUserRole:           {},
	ObserveUserEmailRole:      {},
	ObserveUserFirstNameRole:  {},
	ObserveUserLastNameRole:   {},
	ObserveUserMiddleNameRole: {},
	ObserveUserSessionRole:    {},
	ObserveSessionRole:        {},
	DeleteSessionRole:         {},
	GuestGroupRole:            {},
	UserGroupRole:             {},
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

// Clone creates copy of role.
func (o Role) Clone() Role {
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

// RoleStore represents a role store.
type RoleStore struct {
	baseStore
	roles  map[int64]Role
	byCode map[string]int64
}

// Get returns role by ID.
//
// If there is no role with specified ID then
// sql.ErrNoRows will be returned.
func (s *RoleStore) Get(id int64) (Role, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if role, ok := s.roles[id]; ok {
		return role.Clone(), nil
	}
	return Role{}, sql.ErrNoRows
}

// All returns all roles.
func (s *RoleStore) All() ([]Role, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var roles []Role
	for _, role := range s.roles {
		roles = append(roles, role)
	}
	return roles, nil
}

// GetByCode returns role by code.
//
// If there is no role with specified code then
// sql.ErrNoRows will be returned.
func (s *RoleStore) GetByCode(code string) (Role, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if id, ok := s.byCode[code]; ok {
		if role, ok := s.roles[id]; ok {
			return role.Clone(), nil
		}
	}
	return Role{}, sql.ErrNoRows
}

// CreateTx creates role and returns copy with valid ID.
func (s *RoleStore) CreateTx(tx *sql.Tx, role Role) (Role, error) {
	event, err := s.createObjectEvent(tx, RoleEvent{
		makeBaseEvent(CreateEvent),
		role,
	})
	if err != nil {
		return Role{}, err
	}
	return event.Object().(Role), nil
}

// UpdateTx updates role with specified ID.
func (s *RoleStore) UpdateTx(tx *sql.Tx, role Role) error {
	_, err := s.createObjectEvent(tx, RoleEvent{
		makeBaseEvent(UpdateEvent),
		role,
	})
	return err
}

// DeleteTx deletes role with specified ID.
func (s *RoleStore) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := s.createObjectEvent(tx, RoleEvent{
		makeBaseEvent(DeleteEvent),
		Role{ID: id},
	})
	return err
}

func (s *RoleStore) reset() {
	s.roles = map[int64]Role{}
	s.byCode = map[string]int64{}
}

func (s *RoleStore) onCreateObject(o db.Object) {
	role := o.(Role)
	s.roles[role.ID] = role
	s.byCode[role.Code] = role.ID
}

func (s *RoleStore) onDeleteObject(o db.Object) {
	role := o.(Role)
	delete(s.byCode, role.Code)
	delete(s.roles, role.ID)
}

func (s *RoleStore) onUpdateObject(o db.Object) {
	role := o.(Role)
	if old, ok := s.roles[role.ID]; ok {
		if old.Code != role.Code {
			delete(s.byCode, old.Code)
		}
	}
	s.onCreateObject(o)
}

// NewRoleStore creates a new instance of RoleStore.
func NewRoleStore(
	table, eventTable string, dialect db.Dialect,
) *RoleStore {
	impl := &RoleStore{}
	impl.baseStore = makeBaseStore(
		Role{}, table, RoleEvent{}, eventTable, impl, dialect,
	)
	return impl
}

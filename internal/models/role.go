package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

// Role represents a role.
type Role struct {
	baseObject
	// Name contains role name.
	//
	// Name should be unique for all roles in the events.
	Name string `db:"name"`
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
func (e RoleEvent) Object() Role {
	return e.Role
}

// SetObject sets event role.
func (e *RoleEvent) SetObject(o Role) {
	e.Role = o
}

// RoleStore represents a role store.
type RoleStore struct {
	cachedStore[Role, RoleEvent, *Role, *RoleEvent]
	byName *index[string, Role, *Role]
}

// GetByName returns role by name.
//
// If there is no role with specified name then
// sql.ErrNoRows will be returned.
func (s *RoleStore) GetByName(name string) (Role, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byName.Get(name) {
		if object, ok := s.objects.Get(id); ok {
			return object.Clone(), nil
		}
	}
	return Role{}, sql.ErrNoRows
}

// NewRoleStore creates a new instance of RoleStore.
func NewRoleStore(
	db *gosql.DB, table, eventTable string,
) *RoleStore {
	impl := &RoleStore{
		byName: newIndex(func(o Role) string { return o.Name }),
	}
	impl.cachedStore = makeCachedStore[Role, RoleEvent](
		db, table, eventTable, impl, impl.byName,
	)
	return impl
}

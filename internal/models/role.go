package models

import (
	"context"
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
	byName *btreeIndex[string, Role, *Role]
}

// GetByName returns role by name.
//
// If there is no role with specified name then
// sql.ErrNoRows will be returned.
func (s *RoleStore) GetByName(ctx context.Context, name string) (Role, error) {
	s.mutex.RLock()
	roles := btreeIndexFind(
		s.byName,
		s.objects.Iter(),
		s.mutex.RLocker(),
		name,
	)
	defer func() { _ = roles.Close() }()
	if !roles.Next() {
		if err := roles.Err(); err != nil {
			return Role{}, err
		}
		return Role{}, sql.ErrNoRows
	}
	return roles.Row(), nil
}

// NewRoleStore creates a new instance of RoleStore.
func NewRoleStore(
	db *gosql.DB, table, eventTable string,
) *RoleStore {
	impl := &RoleStore{
		byName: newBTreeIndex(func(o Role) (string, bool) { return o.Name, true }, lessString),
	}
	impl.cachedStore = makeCachedStore[Role, RoleEvent](
		db, table, eventTable, impl, impl.byName,
	)
	return impl
}

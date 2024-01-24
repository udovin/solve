package models

import (
	"github.com/udovin/gosql"
)

// RoleEdge represents connection for roles.
type RoleEdge struct {
	baseObject
	// RoleID contains ID of parent role.
	RoleID int64 `db:"role_id" json:"role_id"`
	// ChildID contains ID of child role.
	ChildID int64 `db:"child_id" json:"child_id"`
}

// Clone creates copy of role edge.
func (o RoleEdge) Clone() RoleEdge {
	return o
}

// RoleEdgeEvent represents role edge event.
type RoleEdgeEvent struct {
	baseEvent
	RoleEdge
}

// Object returns event role edge.
func (e RoleEdgeEvent) Object() RoleEdge {
	return e.RoleEdge
}

// SetObject sets event role edge.
func (e *RoleEdgeEvent) SetObject(o RoleEdge) {
	e.RoleEdge = o
}

// RoleEdgeStore represents a role edge store.
type RoleEdgeStore struct {
	cachedStore[RoleEdge, RoleEdgeEvent, *RoleEdge, *RoleEdgeEvent]
	byRole *index[int64, RoleEdge, *RoleEdge]
}

// FindByRole returns edges by parent ID.
func (s *RoleEdgeStore) FindByRole(id int64) ([]RoleEdge, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []RoleEdge
	for id := range s.byRole.Get(id) {
		if object, ok := s.objects.Get(id); ok {
			objects = append(objects, object.Clone())
		}
	}
	return objects, nil
}

// NewRoleEdgeStore creates a new instance of RoleEdgeStore.
func NewRoleEdgeStore(
	db *gosql.DB, table, eventTable string,
) *RoleEdgeStore {
	impl := &RoleEdgeStore{
		byRole: newIndex(func(o RoleEdge) (int64, bool) { return o.RoleID, true }),
	}
	impl.cachedStore = makeCachedStore[RoleEdge, RoleEdgeEvent](
		db, table, eventTable, impl, impl.byRole,
	)
	return impl
}

package models

import (
	"database/sql"

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
	baseStore[RoleEdge, RoleEdgeEvent, *RoleEdge, *RoleEdgeEvent]
	edges  map[int64]RoleEdge
	byRole index[int64]
}

// Get returns role edge by ID.
//
// If there is no role with specified ID then
// sql.ErrNoRows will be returned.
func (s *RoleEdgeStore) Get(id int64) (RoleEdge, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if edge, ok := s.edges[id]; ok {
		return edge.Clone(), nil
	}
	return RoleEdge{}, sql.ErrNoRows
}

// FindByRole returns edges by parent ID.
func (s *RoleEdgeStore) FindByRole(id int64) ([]RoleEdge, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var edges []RoleEdge
	for id := range s.byRole[id] {
		if edge, ok := s.edges[id]; ok {
			edges = append(edges, edge.Clone())
		}
	}
	return edges, nil
}

func (s *RoleEdgeStore) reset() {
	s.edges = map[int64]RoleEdge{}
	s.byRole = index[int64]{}
}

func (s *RoleEdgeStore) onCreateObject(edge RoleEdge) {
	s.edges[edge.ID] = edge
	s.byRole.Create(edge.RoleID, edge.ID)
}

func (s *RoleEdgeStore) onDeleteObject(id int64) {
	if edge, ok := s.edges[id]; ok {
		s.byRole.Delete(edge.RoleID, edge.ID)
		delete(s.edges, edge.ID)
	}
}

var _ baseStoreImpl[RoleEdge] = (*RoleEdgeStore)(nil)

// NewRoleEdgeStore creates a new instance of RoleEdgeStore.
func NewRoleEdgeStore(
	db *gosql.DB, table, eventTable string,
) *RoleEdgeStore {
	impl := &RoleEdgeStore{}
	impl.baseStore = makeBaseStore[RoleEdge, RoleEdgeEvent](
		db, table, eventTable, impl,
	)
	return impl
}

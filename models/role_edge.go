package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
)

// RoleEdge represents connection for roles.
type RoleEdge struct {
	// ID contains ID of role.
	ID int64 `db:"id" json:"id"`
	// RoleID contains ID of parent role.
	RoleID int64 `db:"role_id" json:"role_id"`
	// ChildID contains ID of child role.
	ChildID int64 `db:"child_id" json:"child_id"`
}

// ObjectID return ID of role edge.
func (o RoleEdge) ObjectID() int64 {
	return o.ID
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
func (e RoleEdgeEvent) Object() db.Object {
	return e.RoleEdge
}

// WithObject returns event with replaced RoleEdge.
func (e RoleEdgeEvent) WithObject(o db.Object) ObjectEvent {
	e.RoleEdge = o.(RoleEdge)
	return e
}

// RoleEdgeStore represents a role edge store.
type RoleEdgeStore struct {
	baseStore
	edges  map[int64]RoleEdge
	byRole indexInt64
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

// CreateTx creates role edge and returns copy with valid ID.
func (s *RoleEdgeStore) CreateTx(
	tx *sql.Tx, edge RoleEdge,
) (RoleEdge, error) {
	event, err := s.createObjectEvent(tx, RoleEdgeEvent{
		makeBaseEvent(CreateEvent),
		edge,
	})
	if err != nil {
		return RoleEdge{}, err
	}
	return event.Object().(RoleEdge), nil
}

// UpdateTx updates role edge with specified ID.
func (s *RoleEdgeStore) UpdateTx(tx *sql.Tx, edge RoleEdge) error {
	_, err := s.createObjectEvent(tx, RoleEdgeEvent{
		makeBaseEvent(UpdateEvent),
		edge,
	})
	return err
}

// DeleteTx deletes role edge with specified ID.
func (s *RoleEdgeStore) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := s.createObjectEvent(tx, RoleEdgeEvent{
		makeBaseEvent(DeleteEvent),
		RoleEdge{ID: id},
	})
	return err
}

func (s *RoleEdgeStore) reset() {
	s.edges = map[int64]RoleEdge{}
	s.byRole = indexInt64{}
}

func (s *RoleEdgeStore) onCreateObject(o db.Object) {
	edge := o.(RoleEdge)
	s.edges[edge.ID] = edge
	s.byRole.Create(edge.RoleID, edge.ID)
}

func (s *RoleEdgeStore) onDeleteObject(o db.Object) {
	edge := o.(RoleEdge)
	s.byRole.Delete(edge.RoleID, edge.ID)
	delete(s.edges, edge.ID)
}

func (s *RoleEdgeStore) onUpdateObject(o db.Object) {
	edge := o.(RoleEdge)
	if old, ok := s.edges[edge.ID]; ok {
		if old.RoleID != edge.RoleID {
			s.byRole.Delete(old.RoleID, old.ID)
		}
	}
	s.onCreateObject(o)
}

// NewRoleEdgeStore creates a new instance of RoleEdgeStore.
func NewRoleEdgeStore(
	table, eventTable string, dialect db.Dialect,
) *RoleEdgeStore {
	impl := &RoleEdgeStore{}
	impl.baseStore = makeBaseStore(
		RoleEdge{}, table, RoleEdgeEvent{}, eventTable, impl, dialect,
	)
	return impl
}

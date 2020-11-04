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

func (o RoleEdge) clone() RoleEdge {
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

// RoleEdgeManager represents a role edge manager.
type RoleEdgeManager struct {
	baseManager
	edges  map[int64]RoleEdge
	byRole indexInt64
}

// Get returns role edge by ID.
//
// If there is no role with specified ID then
// sql.ErrNoRows will be returned.
func (m *RoleEdgeManager) Get(id int64) (RoleEdge, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if edge, ok := m.edges[id]; ok {
		return edge.clone(), nil
	}
	return RoleEdge{}, sql.ErrNoRows
}

// FindByRole returns edges by parent ID.
func (m *RoleEdgeManager) FindByRole(id int64) ([]RoleEdge, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var edges []RoleEdge
	for id := range m.byRole[id] {
		if edge, ok := m.edges[id]; ok {
			edges = append(edges, edge.clone())
		}
	}
	return edges, nil
}

// CreateTx creates role edge and returns copy with valid ID.
func (m *RoleEdgeManager) CreateTx(
	tx *sql.Tx, edge RoleEdge,
) (RoleEdge, error) {
	event, err := m.createObjectEvent(tx, RoleEdgeEvent{
		makeBaseEvent(CreateEvent),
		edge,
	})
	if err != nil {
		return RoleEdge{}, err
	}
	return event.Object().(RoleEdge), nil
}

// UpdateTx updates role edge with specified ID.
func (m *RoleEdgeManager) UpdateTx(tx *sql.Tx, edge RoleEdge) error {
	_, err := m.createObjectEvent(tx, RoleEdgeEvent{
		makeBaseEvent(UpdateEvent),
		edge,
	})
	return err
}

// DeleteTx deletes role edge with specified ID.
func (m *RoleEdgeManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, RoleEdgeEvent{
		makeBaseEvent(DeleteEvent),
		RoleEdge{ID: id},
	})
	return err
}

func (m *RoleEdgeManager) reset() {
	m.edges = map[int64]RoleEdge{}
	m.byRole = indexInt64{}
}

func (m *RoleEdgeManager) onCreateObject(o db.Object) {
	edge := o.(RoleEdge)
	m.edges[edge.ID] = edge
	m.byRole.Create(edge.RoleID, edge.ID)
}

func (m *RoleEdgeManager) onDeleteObject(o db.Object) {
	edge := o.(RoleEdge)
	m.byRole.Delete(edge.RoleID, edge.ID)
	delete(m.edges, edge.ID)
}

func (m *RoleEdgeManager) onUpdateObject(o db.Object) {
	edge := o.(RoleEdge)
	if old, ok := m.edges[edge.ID]; ok {
		if old.RoleID != edge.RoleID {
			m.byRole.Delete(old.RoleID, old.ID)
		}
	}
	m.onCreateObject(o)
}

// NewRoleEdgeManager creates a new instance of RoleEdgeManager.
func NewRoleEdgeManager(
	table, eventTable string, dialect db.Dialect,
) *RoleEdgeManager {
	impl := &RoleEdgeManager{}
	impl.baseManager = makeBaseManager(
		RoleEdge{}, table, RoleEdgeEvent{}, eventTable, impl, dialect,
	)
	return impl
}

package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
)

type Role struct {
	ID       int64  `db:"id" json:""`
	ParentID int64  `db:"parent_id" json:""`
	Code     string `db:"code" json:""`
}

func (o Role) ObjectID() int64 {
	return o.ID
}

type RoleEvent struct {
	baseEvent
	Role
}

func (e RoleEvent) Object() db.Object {
	return e.Role
}

func (e RoleEvent) WithObject(o db.Object) ObjectEvent {
	e.Role = o.(Role)
	return e
}

type RoleManager struct {
	baseManager
}

func (m *RoleManager) reset() {
	panic("implement me")
}

func (m *RoleManager) addObject(o db.Object) {
	panic("implement me")
}

func (m *RoleManager) onCreateObject(o db.Object) {
	panic("implement me")
}

func (m *RoleManager) onUpdateObject(o db.Object) {
	panic("implement me")
}

func (m *RoleManager) onDeleteObject(o db.Object) {
	panic("implement me")
}

func (m *RoleManager) updateSchema(tx *sql.Tx, version int) (int, error) {
	panic("implement me")
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

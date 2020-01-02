package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
)

type UserRole struct {
	Id     int64 `db:"id" json:""`
	RoleId int64 `db:"role_id" json:""`
	UserId int64 `db:"user_id" json:""`
}

func (o UserRole) ObjectId() int64 {
	return o.Id
}

type UserRoleEvent struct {
	baseEvent
	UserRole
}

func (e UserRoleEvent) Object() db.Object {
	return e.UserRole
}

func (e UserRoleEvent) WithObject(o db.Object) ObjectEvent {
	e.UserRole = o.(UserRole)
	return e
}

type UserRoleManager struct {
	baseManager
}

func (m *UserRoleManager) reset() {
	panic("implement me")
}

func (m *UserRoleManager) addObject(o db.Object) {
	panic("implement me")
}

func (m *UserRoleManager) onCreateObject(o db.Object) {
	panic("implement me")
}

func (m *UserRoleManager) onUpdateObject(o db.Object) {
	panic("implement me")
}

func (m *UserRoleManager) onDeleteObject(o db.Object) {
	panic("implement me")
}

func (m *UserRoleManager) migrate(tx *sql.Tx, version int) (int, error) {
	panic("implement me")
}

func NewUserRoleManager(
	table, eventTable string, dialect db.Dialect,
) *UserRoleManager {
	impl := &UserRoleManager{}
	impl.baseManager = makeBaseManager(
		UserRole{}, table, UserRoleEvent{}, eventTable, impl, dialect,
	)
	return impl
}

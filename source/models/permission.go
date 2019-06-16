package models

import (
	"database/sql"
	"fmt"
)

type Permission struct {
	ID   int64  `db:"id"   json:""`
	Code string `db:"code" json:""`
}

type PermissionChange struct {
	Permission
	ID   int64      `db:"change_id"   json:""`
	Type ChangeType `db:"change_type" json:""`
}

type PermissionStore struct {
	db          *sql.DB
	table       string
	changeTable string
	permissions map[int64]Permission
}

func (c *PermissionChange) ChangeID() int64 {
	return c.ID
}

func (c *PermissionChange) ChangeType() ChangeType {
	return c.Type
}

func (c *PermissionChange) ChangeData() interface{} {
	return c.Permission
}

func NewPermissionStore(
	db *sql.DB, table, changeTable string,
) *PermissionStore {
	return &PermissionStore{
		db:          db,
		table:       table,
		changeTable: changeTable,
	}
}

func (s *PermissionStore) GetDB() *sql.DB {
	return s.db
}

func (s *PermissionStore) TableName() string {
	return s.table
}

func (s *PermissionStore) ChangeTableName() string {
	return s.changeTable
}

func (s *PermissionStore) scanChange(scan RowScan) (Change, error) {
	change := &PermissionChange{}
	if err := scan.Scan(change); err != nil {
		return nil, err
	}
	return change, nil
}

func (s *PermissionStore) applyChange(change Change) error {
	permission := change.ChangeData().(Permission)
	switch change.ChangeType() {
	case CreateChange:
		s.permissions[permission.ID] = permission
	case UpdateChange:
		s.permissions[permission.ID] = permission
	case DeleteChange:
		delete(s.permissions, permission.ID)
	default:
		return fmt.Errorf(
			"unsupported change type = %d", change.ChangeType(),
		)
	}
	return nil
}

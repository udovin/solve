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
	Time int64      `db:"change_time" json:""`
}

type PermissionStore struct {
	Manager     *ChangeManager
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

func (c *PermissionChange) ChangeTime() int64 {
	return c.Time
}

func (c *PermissionChange) ChangeData() interface{} {
	return c.Permission
}

func NewPermissionStore(
	db *sql.DB, table, changeTable string,
) *PermissionStore {
	store := PermissionStore{
		db: db, table: table, changeTable: changeTable,
		permissions: make(map[int64]Permission),
	}
	store.Manager = NewChangeManager(&store)
	return &store
}

func (s *PermissionStore) GetDB() *sql.DB {
	return s.db
}

func (s *PermissionStore) ChangeTableName() string {
	return s.changeTable
}

func (s *PermissionStore) scanChange(scan RowScan) (Change, error) {
	change := &PermissionChange{}
	err := scan.Scan(
		&change.ID, &change.Type, &change.Time,
		&change.Permission.ID, &change.Code,
	)
	if err != nil {
		return nil, err
	}
	return change, nil
}

func (s *PermissionStore) createChangeTx(
	tx *sql.Tx, changeType ChangeType, changeTime int64, data interface{},
) (Change, error) {
	var permission Permission
	switch changeType {
	case CreateChange:
		permission = data.(Permission)
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s" ("code") VALUES ($1)`,
				s.table,
			),
			permission.Code,
		)
		if err != nil {
			return nil, err
		}
		permissionID, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		permission.ID = permissionID
	case UpdateChange:
		permission = data.(Permission)
		if _, ok := s.permissions[permission.ID]; !ok {
			return nil, fmt.Errorf(
				"permission with id = %d does not exists", permission.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET "code" = $2 WHERE "id" = $1"`,
				s.table,
			),
			permission.ID, permission.Code,
		)
		if err != nil {
			return nil, err
		}
	case DeleteChange:
		var ok bool
		permission, ok = s.permissions[data.(int64)]
		if !ok {
			return nil, fmt.Errorf(
				"permission with id = %d does not exists", permission.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1"`,
				s.table,
			),
			permission.ID,
		)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf(
			"unsupported change type = %d", changeType,
		)
	}
	res, err := tx.Exec(
		fmt.Sprintf(
			`INSERT INTO "%s" `+
				`("change_type", "change_time", "id", "code") `+
				`VALUES ($1, $2, $3, $4)`,
			s.ChangeTableName(),
		),
		changeType, changeTime, permission.ID, permission.Code,
	)
	if err != nil {
		return nil, err
	}
	changeID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &PermissionChange{
		ID: changeID, Type: changeType,
		Time: changeTime, Permission: permission,
	}, nil
}

func (s *PermissionStore) applyChange(change Change) {
	permission := change.ChangeData().(Permission)
	switch change.ChangeType() {
	case CreateChange:
		s.permissions[permission.ID] = permission
	case UpdateChange:
		s.permissions[permission.ID] = permission
	case DeleteChange:
		delete(s.permissions, permission.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %d", change.ChangeType(),
		))
	}
}

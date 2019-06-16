package models

import (
	"database/sql"
	"fmt"
)

type Role struct {
	ID   int64  `db:"id"   json:""`
	Code string `db:"code" json:""`
}

type RoleChange struct {
	Role
	ID   int64      `db:"change_id"   json:""`
	Type ChangeType `db:"change_type" json:""`
	Time int64      `db:"change_time" json:""`
}

type RoleStore struct {
	Manager     *ChangeManager
	db          *sql.DB
	table       string
	changeTable string
	roles       map[int64]Role
}

func (c *RoleChange) ChangeID() int64 {
	return c.ID
}

func (c *RoleChange) ChangeType() ChangeType {
	return c.Type
}

func (c *RoleChange) ChangeTime() int64 {
	return c.Time
}

func (c *RoleChange) ChangeData() interface{} {
	return c.Role
}

func NewRoleStore(
	db *sql.DB, table, changeTable string,
) *RoleStore {
	store := RoleStore{
		db: db, table: table, changeTable: changeTable,
		roles: make(map[int64]Role),
	}
	store.Manager = NewChangeManager(&store)
	return &store
}

func (s *RoleStore) GetDB() *sql.DB {
	return s.db
}

func (s *RoleStore) ChangeTableName() string {
	return s.changeTable
}

func (s *RoleStore) scanChange(scan RowScan) (Change, error) {
	change := &RoleChange{}
	err := scan.Scan(
		&change.ID, &change.Type, &change.Time,
		&change.Role.ID, &change.Code,
	)
	if err != nil {
		return nil, err
	}
	return change, nil
}

func (s *RoleStore) createChangeTx(
	tx *sql.Tx, changeType ChangeType, changeTime int64, data interface{},
) (Change, error) {
	var role Role
	switch changeType {
	case CreateChange:
		role = data.(Role)
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s" ("code") VALUES ($1)`,
				s.table,
			),
			role.Code,
		)
		if err != nil {
			return nil, err
		}
		roleID, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		role.ID = roleID
	case UpdateChange:
		role = data.(Role)
		if _, ok := s.roles[role.ID]; !ok {
			return nil, fmt.Errorf(
				"role with id = %d does not exists", role.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET "code" = $1 WHERE "id" = $2"`,
				s.table,
			),
			role.Code, role.ID,
		)
		if err != nil {
			return nil, err
		}
	case DeleteChange:
		var ok bool
		role, ok = s.roles[data.(int64)]
		if !ok {
			return nil, fmt.Errorf(
				"role with id = %d does not exists", role.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1"`,
				s.table,
			),
			role.ID,
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
		changeType, changeTime, role.ID, role.Code,
	)
	if err != nil {
		return nil, err
	}
	changeID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &RoleChange{
		ID: changeID, Type: changeType,
		Time: changeTime, Role: role,
	}, nil
}

func (s *RoleStore) applyChange(change Change) {
	role := change.ChangeData().(Role)
	switch change.ChangeType() {
	case CreateChange:
		s.roles[role.ID] = role
	case UpdateChange:
		s.roles[role.ID] = role
	case DeleteChange:
		delete(s.roles, role.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %d", change.ChangeType(),
		))
	}
}

func (s *RoleStore) Get(id int64) (Role, error) {
	role, ok := s.roles[id]
	if !ok {
		return role, sql.ErrNoRows
	}
	return role, nil
}

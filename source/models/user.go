package models

import (
	"database/sql"
	"fmt"
)

type User struct {
	ID           int64  `db:"id"          json:""`
	Login        string `db:"login"       json:""`
	CreateTime   int64  `db:"create_time" json:""`
	PasswordHash string `db:"password_hash"`
	PasswordSalt string `db:"password_salt"`
}

type UserChange struct {
	User
	ID   int64      `db:"change_id"   json:""`
	Type ChangeType `db:"change_type" json:""`
}

type UserStore struct {
	db          *sql.DB
	table       string
	changeTable string
	users       map[int64]User
}

func (c *UserChange) ChangeID() int64 {
	return c.ID
}

func (c *UserChange) ChangeType() ChangeType {
	return c.Type
}

func (c *UserChange) ChangeData() interface{} {
	return c.User
}

func NewUserStore(
	db *sql.DB, table, changeTable string,
) *UserStore {
	return &UserStore{
		db:          db,
		table:       table,
		changeTable: changeTable,
	}
}

func (s *UserStore) GetDB() *sql.DB {
	return s.db
}

func (s *UserStore) TableName() string {
	return s.table
}

func (s *UserStore) ChangeTableName() string {
	return s.changeTable
}

func (s *UserStore) scanChange(scan RowScan) (Change, error) {
	change := &UserChange{}
	if err := scan.Scan(change); err != nil {
		return nil, err
	}
	return change, nil
}

func (s *UserStore) applyChange(change Change) error {
	user := change.ChangeData().(User)
	switch change.ChangeType() {
	case CreateChange:
		s.users[user.ID] = user
	case UpdateChange:
		s.users[user.ID] = user
	case DeleteChange:
		delete(s.users, user.ID)
	default:
		return fmt.Errorf(
			"unsupported change type = %d", change.ChangeType(),
		)
	}
	return nil
}

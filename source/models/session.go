package models

import (
	"database/sql"
	"fmt"
)

type Session struct {
	ID         int64  `db:"id"          json:""`
	UserID     int64  `db:"user_id"     json:""`
	Secret     string `db:"secret"      json:""`
	CreateTime int64  `db:"create_time" json:""`
}

type SessionChange struct {
	Session
	ID   int64      `db:"change_id"   json:""`
	Type ChangeType `db:"change_type" json:""`
}

type SessionStore struct {
	db          *sql.DB
	table       string
	changeTable string
	sessions    map[int64]Session
}

func (c *SessionChange) ChangeID() int64 {
	return c.ID
}

func (c *SessionChange) ChangeType() ChangeType {
	return c.Type
}

func (c *SessionChange) ChangeData() interface{} {
	return c.Session
}

func NewSessionStore(
	db *sql.DB, table, changeTable string,
) *SessionStore {
	return &SessionStore{
		db:          db,
		table:       table,
		changeTable: changeTable,
	}
}

func (s *SessionStore) GetDB() *sql.DB {
	return s.db
}

func (s *SessionStore) TableName() string {
	return s.table
}

func (s *SessionStore) ChangeTableName() string {
	return s.changeTable
}

func (s *SessionStore) scanChange(scan RowScan) (Change, error) {
	change := &SessionChange{}
	if err := scan.Scan(change); err != nil {
		return nil, err
	}
	return change, nil
}

func (s *SessionStore) applyChange(change Change) error {
	session := change.ChangeData().(Session)
	switch change.ChangeType() {
	case CreateChange:
		s.sessions[session.ID] = session
	case UpdateChange:
		s.sessions[session.ID] = session
	case DeleteChange:
		delete(s.sessions, session.ID)
	default:
		return fmt.Errorf(
			"unsupported change type = %d", change.ChangeType(),
		)
	}
	return nil
}

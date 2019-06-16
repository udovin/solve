package models

import (
	"database/sql"
	"fmt"
)

type Session struct {
	ID         int64  `db:"id"          json:""`
	UserID     int64  `db:"user_id"     json:""`
	Secret     string `db:"secret"      json:"-"`
	CreateTime int64  `db:"create_time" json:""`
	ExpireTime int64  `db:"expire_time" json:""`
}

type SessionChange struct {
	Session
	ID   int64      `db:"change_id"   json:""`
	Type ChangeType `db:"change_type" json:""`
	Time int64      `db:"change_time" json:""`
}

type SessionStore struct {
	Manager     *ChangeManager
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

func (c *SessionChange) ChangeTime() int64 {
	return c.Time
}

func (c *SessionChange) ChangeData() interface{} {
	return c.Session
}

func NewSessionStore(
	db *sql.DB, table, changeTable string,
) *SessionStore {
	store := SessionStore{
		db: db, table: table, changeTable: changeTable,
		sessions: make(map[int64]Session),
	}
	store.Manager = NewChangeManager(&store)
	return &store
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
	err := scan.Scan(
		&change.ID, &change.Type, &change.Time,
		&change.Session.ID, &change.UserID,
		&change.Secret, &change.CreateTime,
		&change.ExpireTime,
	)
	if err != nil {
		return nil, err
	}
	return change, nil
}

func (s *SessionStore) createChangeTx(
	tx *sql.Tx, changeType ChangeType, changeTime int64, data interface{},
) (Change, error) {
	var session Session
	switch changeType {
	case CreateChange:
		session = data.(Session)
		session.CreateTime = changeTime
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s" `+
					`("user_id", "secret", "create_time", "expire_time") `+
					`VALUES ($1, $2, $3, $4)`,
				s.table,
			),
			session.UserID, session.Secret,
			session.CreateTime, session.ExpireTime,
		)
		if err != nil {
			return nil, err
		}
		sessionID, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		session.ID = sessionID
	case UpdateChange:
		session = data.(Session)
		if _, ok := s.sessions[session.ID]; !ok {
			return nil, fmt.Errorf(
				"session with id = %d does not exists", session.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET `+
					`'"user_id" = $2, "secret" = $3, "expire_time" = $4`+
					`WHERE "id" = $1"`,
				s.table,
			),
			session.ID, session.UserID, session.Secret, session.ExpireTime,
		)
		if err != nil {
			return nil, err
		}
	case DeleteChange:
		var ok bool
		session, ok = s.sessions[data.(int64)]
		if !ok {
			return nil, fmt.Errorf(
				"session with id = %d does not exists", session.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1"`,
				s.table,
			),
			session.ID,
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
				`("change_type", "change_time", "id", "user_id", `+
				`"secret", "create_time", "expire_time") `+
				`VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			s.ChangeTableName(),
		),
		changeType, changeTime, session.ID, session.UserID,
		session.Secret, session.CreateTime, session.ExpireTime,
	)
	if err != nil {
		return nil, err
	}
	changeID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &SessionChange{
		ID: changeID, Type: changeType,
		Time: changeTime, Session: session,
	}, nil
}

func (s *SessionStore) applyChange(change Change) {
	session := change.ChangeData().(Session)
	switch change.ChangeType() {
	case CreateChange:
		s.sessions[session.ID] = session
	case UpdateChange:
		s.sessions[session.ID] = session
	case DeleteChange:
		delete(s.sessions, session.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %d", change.ChangeType(),
		))
	}
}

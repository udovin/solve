package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"
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
	ChangeBase
}

type SessionStore struct {
	Manager     *ChangeManager
	db          *sql.DB
	table       string
	changeTable string
	sessions    map[int64]Session
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

func (s *SessionStore) Create(m *Session) error {
	change := SessionChange{
		ChangeBase: ChangeBase{Type: CreateChange},
		Session:    *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Session
	return nil
}

func (s *SessionStore) Update(m *Session) error {
	change := SessionChange{
		ChangeBase: ChangeBase{Type: UpdateChange},
		Session:    *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Session
	return nil
}

func (s *SessionStore) Delete(id int64) error {
	change := SessionChange{
		ChangeBase: ChangeBase{Type: DeleteChange},
		Session:    Session{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *SessionStore) scanChange(scan RowScan) (Change, error) {
	change := &SessionChange{}
	err := scan.Scan(
		&change.ChangeBase.ID, &change.Type, &change.Time,
		&change.Session.ID, &change.UserID,
		&change.Secret, &change.CreateTime,
		&change.ExpireTime,
	)
	if err != nil {
		return nil, err
	}
	return change, nil
}

func (s *SessionStore) saveChangeTx(tx *sql.Tx, change Change) error {
	session := change.(*SessionChange)
	session.Time = time.Now().Unix()
	switch session.Type {
	case CreateChange:
		session.CreateTime = session.Time
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
			return err
		}
		session.Session.ID, err = res.LastInsertId()
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.sessions[session.Session.ID]; !ok {
			return fmt.Errorf(
				"session with id = %d does not exists",
				session.Session.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET `+
					`'"user_id" = $1, "secret" = $2, "expire_time" = $3`+
					`WHERE "id" = $4`,
				s.table,
			),
			session.UserID, session.Secret,
			session.ExpireTime, session.Session.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.sessions[session.Session.ID]; !ok {
			return fmt.Errorf(
				"session with id = %d does not exists",
				session.Session.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1`,
				s.table,
			),
			session.Session.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			session.Type,
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
		session.Type, session.Time, session.Session.ID, session.UserID,
		session.Secret, session.CreateTime, session.ExpireTime,
	)
	if err != nil {
		return err
	}
	session.ChangeBase.ID, err = res.LastInsertId()
	return err
}

func (s *SessionStore) applyChange(change Change) {
	sessionChange := change.(*SessionChange)
	session := sessionChange.Session
	switch sessionChange.Type {
	case CreateChange:
		s.sessions[session.ID] = session
	case UpdateChange:
		s.sessions[session.ID] = session
	case DeleteChange:
		delete(s.sessions, session.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			sessionChange.Type,
		))
	}
}

func (m *Session) GenerateSecret() error {
	secretBytes := make([]byte, 40)
	if _, err := rand.Read(secretBytes); err != nil {
		return err
	}
	m.Secret = base64.StdEncoding.EncodeToString(secretBytes)
	return nil
}

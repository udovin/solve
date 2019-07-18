package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
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
	BaseChange
	Session
}

type SessionStore struct {
	Manager     *ChangeManager
	db          *sql.DB
	table       string
	changeTable string
	sessions    map[int64]Session
	userMap     map[int64]map[int64]struct{}
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
		userMap:  make(map[int64]map[int64]struct{}),
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

func (s *SessionStore) Get(id int64) (Session, bool) {
	session, ok := s.sessions[id]
	return session, ok
}

func (s *SessionStore) GetByUser(userID int64) []Session {
	if idSet, ok := s.userMap[userID]; ok {
		var sessions []Session
		for id := range idSet {
			if session, ok := s.sessions[id]; ok {
				sessions = append(sessions, session)
			}
		}
		return sessions
	}
	return nil
}

func (s *SessionStore) GetByCookie(cookie string) (Session, bool) {
	parts := strings.SplitN(cookie, "_", 2)
	id, err := strconv.ParseInt(parts[0], 10, 60)
	if err != nil {
		return Session{}, false
	}
	session, ok := s.sessions[id]
	if !ok || session.Secret != parts[1] {
		return Session{}, false
	}
	return session, true
}

func (s *SessionStore) Create(m *Session) error {
	change := SessionChange{
		BaseChange: BaseChange{Type: CreateChange},
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
		BaseChange: BaseChange{Type: UpdateChange},
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
		BaseChange: BaseChange{Type: DeleteChange},
		Session:    Session{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *SessionStore) loadChangeGapTx(
	tx *ChangeTx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time", "id",`+
				` "user_id", "secret", "create_time", "expire_time"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.ChangeTableName(),
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *SessionStore) scanChange(scan Scanner) (Change, error) {
	change := &SessionChange{}
	err := scan.Scan(
		&change.BaseChange.ID, &change.Type, &change.Time,
		&change.Session.ID, &change.UserID, &change.Secret,
		&change.CreateTime, &change.ExpireTime,
	)
	return change, err
}

func (s *SessionStore) saveChangeTx(tx *ChangeTx, change Change) error {
	session := change.(*SessionChange)
	session.Time = time.Now().Unix()
	switch session.Type {
	case CreateChange:
		session.CreateTime = session.Time
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("user_id", "secret", "create_time", "expire_time")`+
					` VALUES ($1, $2, $3, $4)`,
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
				`UPDATE "%s" SET`+
					` "user_id" = $1, "secret" = $2, "expire_time" = $3`+
					` WHERE "id" = $4`,
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
			`INSERT INTO "%s"`+
				` ("change_type", "change_time", "id", "user_id",`+
				` "secret", "create_time", "expire_time")`+
				` VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			s.ChangeTableName(),
		),
		session.Type, session.Time, session.Session.ID, session.UserID,
		session.Secret, session.CreateTime, session.ExpireTime,
	)
	if err != nil {
		return err
	}
	session.BaseChange.ID, err = res.LastInsertId()
	return err
}

func (s *SessionStore) applyChange(change Change) {
	sessionChange := change.(*SessionChange)
	session := sessionChange.Session
	switch sessionChange.Type {
	case UpdateChange:
		if oldSession, ok := s.sessions[session.ID]; ok {
			if oldSession.UserID != session.UserID {
				delete(s.userMap[oldSession.UserID], oldSession.ID)
				if len(s.userMap[oldSession.UserID]) == 0 {
					delete(s.userMap, oldSession.UserID)
				}
			}
		}
		fallthrough
	case CreateChange:
		if s.userMap[session.UserID] == nil {
			s.userMap[session.UserID] = make(map[int64]struct{})
		}
		s.userMap[session.UserID][session.ID] = struct{}{}
		s.sessions[session.ID] = session
	case DeleteChange:
		if s.userMap[session.UserID] != nil {
			delete(s.userMap[session.UserID], session.ID)
			if len(s.userMap[session.UserID]) == 0 {
				delete(s.userMap, session.UserID)
			}
		}
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

func (m *Session) FormatCookie() string {
	return fmt.Sprintf("%d_%s", m.ID, m.Secret)
}

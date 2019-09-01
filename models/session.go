package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Session struct {
	ID         int64  `json:""  db:"id"`
	UserID     int64  `json:""  db:"user_id"`
	Secret     string `json:"-" db:"secret"`
	CreateTime int64  `json:""  db:"create_time"`
	ExpireTime int64  `json:""  db:"expire_time"`
}

type sessionChange struct {
	BaseChange
	Session
}

type SessionStore struct {
	Manager      *ChangeManager
	db           *sql.DB
	table        string
	changeTable  string
	sessions     map[int64]Session
	userSessions map[int64]map[int64]struct{}
	mutex        sync.RWMutex
}

func (c *sessionChange) ChangeData() interface{} {
	return c.Session
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

func NewSessionStore(db *sql.DB, table, changeTable string) *SessionStore {
	store := SessionStore{
		table:        table,
		changeTable:  changeTable,
		sessions:     make(map[int64]Session),
		userSessions: make(map[int64]map[int64]struct{}),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *SessionStore) Get(id int64) (Session, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	session, ok := s.sessions[id]
	return session, ok
}

func (s *SessionStore) GetByUser(userID int64) []Session {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if idSet, ok := s.userSessions[userID]; ok {
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
	s.mutex.RLock()
	defer s.mutex.RUnlock()
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
	change := sessionChange{
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
	change := sessionChange{
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
	change := sessionChange{
		BaseChange: BaseChange{Type: DeleteChange},
		Session:    Session{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *SessionStore) GetLocker() sync.Locker {
	return &s.mutex
}

func (s *SessionStore) InitChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *SessionStore) LoadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time", "id",`+
				` "user_id", "secret", "create_time", "expire_time"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *SessionStore) ScanChange(scan Scanner) (Change, error) {
	session := sessionChange{}
	err := scan.Scan(
		&session.BaseChange.ID, &session.Type, &session.Time,
		&session.Session.ID, &session.UserID, &session.Secret,
		&session.CreateTime, &session.ExpireTime,
	)
	return &session, err
}

func (s *SessionStore) SaveChange(tx *sql.Tx, change Change) error {
	session := change.(*sessionChange)
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
			s.changeTable,
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

func (s *SessionStore) ApplyChange(change Change) {
	session := change.(*sessionChange)
	switch session.Type {
	case UpdateChange:
		if oldSession, ok := s.sessions[session.Session.ID]; ok {
			if oldSession.UserID != session.UserID {
				if userSessions, ok := s.userSessions[oldSession.UserID]; ok {
					delete(userSessions, oldSession.ID)
					if len(userSessions) == 0 {
						delete(s.userSessions, oldSession.UserID)
					}
				}
			}
		}
		fallthrough
	case CreateChange:
		if _, ok := s.userSessions[session.UserID]; !ok {
			s.userSessions[session.UserID] = make(map[int64]struct{})
		}
		s.userSessions[session.UserID][session.Session.ID] = struct{}{}
		s.sessions[session.Session.ID] = session.Session
	case DeleteChange:
		if userSessions, ok := s.userSessions[session.UserID]; ok {
			delete(userSessions, session.Session.ID)
			if len(userSessions) == 0 {
				delete(s.userSessions, session.UserID)
			}
		}
		delete(s.sessions, session.Session.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			session.Type,
		))
	}
}

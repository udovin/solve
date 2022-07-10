package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/udovin/gosql"
)

// Session represents account session.
type Session struct {
	baseObject
	// AccountID contains ID of account.
	AccountID int64 `db:"account_id"`
	// Secret contains secret string of session.
	Secret string `db:"secret"`
	// CreateTime contains time when session was created.
	CreateTime int64 `db:"create_time"`
	// ExpireTime contains time when session should be expired.
	ExpireTime int64 `db:"expire_time"`
	// RemoteAddr contains remote address of user for created session.
	RemoteAddr string `db:"remote_addr"`
	// UserAgent contains user agent header for created session.
	UserAgent string `db:"user_agent"`
}

// Clone creates copy of session.
func (o Session) Clone() Session {
	return o
}

// GenerateSecret generates a new value for session secret.
func (o *Session) GenerateSecret() error {
	bytes := make([]byte, 40)
	if _, err := rand.Read(bytes); err != nil {
		return err
	}
	o.Secret = base64.StdEncoding.EncodeToString(bytes)
	return nil
}

// Cookie returns cookie object.
func (o Session) Cookie() http.Cookie {
	return http.Cookie{
		Value:   fmt.Sprintf("%d_%s", o.ID, o.Secret),
		Expires: time.Unix(o.ExpireTime, 0),
	}
}

// SessionEvent represents session event.
type SessionEvent struct {
	baseEvent
	Session
}

// Object returns session.
func (e SessionEvent) Object() Session {
	return e.Session
}

// SetObject sets event session.
func (e *SessionEvent) SetObject(o Session) {
	e.Session = o
}

// SessionStore represents store for sessions.
type SessionStore struct {
	baseStore[Session, SessionEvent, *Session, *SessionEvent]
	sessions  map[int64]Session
	byAccount index[int64]
}

// Get returns session by session ID.
func (s *SessionStore) Get(id int64) (Session, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if session, ok := s.sessions[id]; ok {
		return session.Clone(), nil
	}
	return Session{}, sql.ErrNoRows
}

// FindByAccount returns sessions by account ID.
func (s *SessionStore) FindByAccount(id int64) ([]Session, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var sessions []Session
	for id := range s.byAccount[id] {
		if session, ok := s.sessions[id]; ok {
			sessions = append(sessions, session.Clone())
		}
	}
	return sessions, nil
}

// GetByCookie returns session for specified cookie value.
func (s *SessionStore) GetByCookie(cookie string) (Session, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	parts := strings.SplitN(cookie, "_", 2)
	id, err := strconv.ParseInt(parts[0], 10, 60)
	if err != nil {
		return Session{}, err
	}
	session, ok := s.sessions[id]
	if !ok || session.Secret != parts[1] {
		return Session{}, sql.ErrNoRows
	}
	return session.Clone(), nil
}

func (s *SessionStore) reset() {
	s.sessions = map[int64]Session{}
	s.byAccount = index[int64]{}
}

func (s *SessionStore) onCreateObject(session Session) {
	s.sessions[session.ID] = session
	s.byAccount.Create(session.AccountID, session.ID)
}

func (s *SessionStore) onDeleteObject(id int64) {
	if session, ok := s.sessions[id]; ok {
		s.byAccount.Delete(session.AccountID, session.ID)
		delete(s.sessions, session.ID)
	}
}

var _ baseStoreImpl[Session] = (*SessionStore)(nil)

// NewSessionStore creates a new instance of SessionStore.
func NewSessionStore(
	db *gosql.DB, table, eventTable string,
) *SessionStore {
	impl := &SessionStore{}
	impl.baseStore = makeBaseStore[Session, SessionEvent](
		db, table, eventTable, impl,
	)
	return impl
}

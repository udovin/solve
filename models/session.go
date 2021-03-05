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

	"github.com/udovin/solve/db"
)

// Session represents account session.
type Session struct {
	// ID contains ID of session.
	ID int64 `db:"id" json:""`
	// AccountID contains ID of account.
	AccountID int64 `db:"account_id" json:"-"`
	// Secret contains secret string of session.
	Secret string `db:"secret" json:"-"`
	// CreateTime contains time when session was created.
	CreateTime int64 `db:"create_time" json:""`
	// ExpireTime contains time when session should be expired.
	ExpireTime int64 `db:"expire_time" json:""`
}

// ObjectID returns session ID.
func (o Session) ObjectID() int64 {
	return o.ID
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
func (e SessionEvent) Object() db.Object {
	return e.Session
}

// WithObject returns copy of event with replaced session.
func (e SessionEvent) WithObject(o db.Object) ObjectEvent {
	e.Session = o.(Session)
	return e
}

// SessionStore represents store for sessions.
type SessionStore struct {
	baseStore
	sessions  map[int64]Session
	byAccount indexInt64
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

// CreateTx creates session and returns new session with valid ID.
func (s *SessionStore) CreateTx(
	tx *sql.Tx, session Session,
) (Session, error) {
	event, err := s.createObjectEvent(tx, SessionEvent{
		makeBaseEvent(CreateEvent),
		session,
	})
	if err != nil {
		return Session{}, err
	}
	return event.Object().(Session), nil
}

// UpdateTx updates session with specified ID.
func (s *SessionStore) UpdateTx(tx *sql.Tx, session Session) error {
	_, err := s.createObjectEvent(tx, SessionEvent{
		makeBaseEvent(UpdateEvent),
		session,
	})
	return err
}

// DeleteTx deletes session with specified ID.
func (s *SessionStore) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := s.createObjectEvent(tx, SessionEvent{
		makeBaseEvent(DeleteEvent),
		Session{ID: id},
	})
	return err
}

func (s *SessionStore) reset() {
	s.sessions = map[int64]Session{}
	s.byAccount = indexInt64{}
}

func (s *SessionStore) onCreateObject(o db.Object) {
	session := o.(Session)
	s.sessions[session.ID] = session
	s.byAccount.Create(session.AccountID, session.ID)
}

func (s *SessionStore) onDeleteObject(o db.Object) {
	session := o.(Session)
	s.byAccount.Delete(session.AccountID, session.ID)
	delete(s.sessions, session.ID)
}

func (s *SessionStore) onUpdateObject(o db.Object) {
	session := o.(Session)
	if old, ok := s.sessions[session.ID]; ok {
		if old.AccountID != session.AccountID {
			s.byAccount.Delete(old.AccountID, old.ID)
		}
	}
	s.onCreateObject(o)
}

// NewSessionStore creates a new instance of SessionStore.
func NewSessionStore(
	table, eventTable string, dialect db.Dialect,
) *SessionStore {
	impl := &SessionStore{}
	impl.baseStore = makeBaseStore(
		Session{}, table, SessionEvent{}, eventTable, impl, dialect,
	)
	return impl
}

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

// Session represents user session.
type Session struct {
	// ID contains ID of session.
	ID int64 `db:"id" json:""`
	// UserID contains ID of user.
	UserID int64 `db:"user_id" json:""`
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

// SessionManager represents manager for sessions.
type SessionManager struct {
	baseManager
	sessions map[int64]Session
	byUser   indexInt64
}

// Get returns session by session ID.
func (m *SessionManager) Get(id int64) (Session, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if session, ok := m.sessions[id]; ok {
		return session, nil
	}
	return Session{}, sql.ErrNoRows
}

// FindByUser returns sessions by user ID.
func (m *SessionManager) FindByUser(userID int64) ([]Session, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var sessions []Session
	for id := range m.byUser[userID] {
		if session, ok := m.sessions[id]; ok {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

// GetByCookie returns session for specified cookie value.
func (m *SessionManager) GetByCookie(cookie string) (Session, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	parts := strings.SplitN(cookie, "_", 2)
	id, err := strconv.ParseInt(parts[0], 10, 60)
	if err != nil {
		return Session{}, err
	}
	session, ok := m.sessions[id]
	if !ok || session.Secret != parts[1] {
		return Session{}, sql.ErrNoRows
	}
	return session, nil
}

// CreateTx creates session and returns new session with valid ID.
func (m *SessionManager) CreateTx(
	tx *sql.Tx, session Session,
) (Session, error) {
	event, err := m.createObjectEvent(tx, SessionEvent{
		makeBaseEvent(CreateEvent),
		session,
	})
	if err != nil {
		return Session{}, err
	}
	return event.Object().(Session), nil
}

// UpdateTx updates session with specified ID.
func (m *SessionManager) UpdateTx(tx *sql.Tx, session Session) error {
	_, err := m.createObjectEvent(tx, SessionEvent{
		makeBaseEvent(UpdateEvent),
		session,
	})
	return err
}

// DeleteTx deletes session with specified ID.
func (m *SessionManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, SessionEvent{
		makeBaseEvent(DeleteEvent),
		Session{ID: id},
	})
	return err
}

func (m *SessionManager) reset() {
	m.sessions = map[int64]Session{}
	m.byUser = indexInt64{}
}

func (m *SessionManager) onCreateObject(o db.Object) {
	session := o.(Session)
	m.sessions[session.ID] = session
	m.byUser.Create(session.UserID, session.ID)
}

func (m *SessionManager) onDeleteObject(o db.Object) {
	session := o.(Session)
	m.byUser.Delete(session.UserID, session.ID)
	delete(m.sessions, session.ID)
}

func (m *SessionManager) onUpdateObject(o db.Object) {
	session := o.(Session)
	if old, ok := m.sessions[session.ID]; ok {
		if old.UserID != session.UserID {
			m.byUser.Delete(old.UserID, old.ID)
		}
	}
	m.onCreateObject(o)
}

// NewSessionManager creates a new instance of SessionManager.
func NewSessionManager(
	table, eventTable string, dialect db.Dialect,
) *SessionManager {
	impl := &SessionManager{}
	impl.baseManager = makeBaseManager(
		Session{}, table, SessionEvent{}, eventTable, impl, dialect,
	)
	return impl
}

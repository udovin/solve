package models

import (
	"context"
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
	// RealIP contains real IP of user for created session.
	RealIP string `db:"real_ip"`
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
	cachedStore[Session, SessionEvent, *Session, *SessionEvent]
	byAccount *index[int64, Session, *Session]
}

// FindByAccount returns sessions by account ID.
func (s *SessionStore) FindByAccount(id int64) ([]Session, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []Session
	for id := range s.byAccount.Get(id) {
		if object, ok := s.objects.Get(id); ok {
			objects = append(objects, object.Clone())
		}
	}
	return objects, nil
}

// GetByCookie returns session for specified cookie value.
func (s *SessionStore) GetByCookie(cookie string) (Session, error) {
	parts := strings.SplitN(cookie, "_", 2)
	if len(parts) != 2 {
		return Session{}, fmt.Errorf("invalid cookie")
	}
	id, err := strconv.ParseInt(parts[0], 10, 60)
	if err != nil {
		return Session{}, err
	}
	session, err := s.Get(context.Background(), id)
	if err != nil {
		return Session{}, err
	}
	if session.Secret != parts[1] {
		return Session{}, sql.ErrNoRows
	}
	return session, nil
}

// NewSessionStore creates a new instance of SessionStore.
func NewSessionStore(
	db *gosql.DB, table, eventTable string,
) *SessionStore {
	impl := &SessionStore{
		byAccount: newIndex(func(o Session) int64 { return o.AccountID }),
	}
	impl.cachedStore = makeCachedStore[Session, SessionEvent](
		db, table, eventTable, impl, impl.byAccount,
	)
	return impl
}

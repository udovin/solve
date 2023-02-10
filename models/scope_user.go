package models

import (
	"crypto/rand"
	"database/sql"
	"strings"

	"github.com/udovin/gosql"
)

// ScopeUser contains common information about scope user.
type ScopeUser struct {
	baseObject
	AccountID    int64   `db:"account_id"`
	ScopeID      int64   `db:"scope_id"`
	Login        string  `db:"login"`
	PasswordHash string  `db:"password_hash"`
	PasswordSalt string  `db:"password_salt"`
	Title        NString `db:"title"`
}

// AccountKind returns ScopeUserAccount kind.
func (o ScopeUser) AccountKind() AccountKind {
	return ScopeUserAccount
}

// Clone creates copy of scope user.
func (o ScopeUser) Clone() ScopeUser {
	return o
}

// ScopeUserEvent represents an scope user event.
type ScopeUserEvent struct {
	baseEvent
	ScopeUser
}

// Object returns scope user.
func (e ScopeUserEvent) Object() ScopeUser {
	return e.ScopeUser
}

// SetObject sets event scope user.
func (e *ScopeUserEvent) SetObject(o ScopeUser) {
	e.ScopeUser = o
}

// ScopeUserStore represents scope users store.
type ScopeUserStore struct {
	cachedStore[ScopeUser, ScopeUserEvent, *ScopeUser, *ScopeUserEvent]
	byAccount    *index[int64, ScopeUser, *ScopeUser]
	byScope      *index[int64, ScopeUser, *ScopeUser]
	byScopeLogin *index[pair[int64, string], ScopeUser, *ScopeUser]
	salt         string
}

// FindByScope returns scope users by scope.
func (s *ScopeUserStore) FindByScope(scope int64) ([]ScopeUser, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []ScopeUser
	for id := range s.byScope.Get(scope) {
		if object, ok := s.objects.Get(id); ok {
			objects = append(objects, object)
		}
	}
	return objects, nil
}

// GetByScopeLogin returns scope user by scope and login.
func (s *ScopeUserStore) GetByScopeLogin(scope int64, login string) (ScopeUser, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byScopeLogin.Get(makePair(scope, strings.ToLower(login))) {
		if object, ok := s.objects.Get(id); ok {
			return object.Clone(), nil
		}
	}
	return ScopeUser{}, sql.ErrNoRows
}

// GetByAccount returns scope user by account id.
func (s *ScopeUserStore) GetByAccount(id int64) (ScopeUser, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byAccount.Get(id) {
		if object, ok := s.objects.Get(id); ok {
			return object.Clone(), nil
		}
	}
	return ScopeUser{}, sql.ErrNoRows
}

// SetPassword modifies PasswordHash and PasswordSalt fields.
//
// PasswordSalt will be replaced with random 16 byte string
// and PasswordHash will be calculated using password, salt
// and global salt.
func (s *ScopeUserStore) SetPassword(user *ScopeUser, password string) error {
	saltBytes := make([]byte, 16)
	_, err := rand.Read(saltBytes)
	if err != nil {
		return err
	}
	user.PasswordSalt = encodeBase64(saltBytes)
	user.PasswordHash = hashPassword(password, user.PasswordSalt, s.salt)
	return nil
}

// CheckPassword checks that passwords are the same.
func (s *ScopeUserStore) CheckPassword(user ScopeUser, password string) bool {
	passwordHash := hashPassword(password, user.PasswordSalt, s.salt)
	return passwordHash == user.PasswordHash
}

// NewScopeUserStore creates new instance of scope user store.
func NewScopeUserStore(
	db *gosql.DB, table, eventTable, salt string,
) *ScopeUserStore {
	impl := &ScopeUserStore{
		byAccount: newIndex(func(o ScopeUser) int64 { return o.AccountID }),
		byScope:   newIndex(func(o ScopeUser) int64 { return o.ScopeID }),
		byScopeLogin: newIndex(func(o ScopeUser) pair[int64, string] {
			return makePair(o.ScopeID, strings.ToLower(o.Login))
		}),
		salt: salt,
	}
	impl.cachedStore = makeCachedStore[ScopeUser, ScopeUserEvent](
		db, table, eventTable, impl, impl.byAccount, impl.byScope, impl.byScopeLogin,
	)
	return impl
}

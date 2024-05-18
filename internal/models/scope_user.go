package models

import (
	"crypto/rand"
	"strings"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
)

// ScopeUser contains common information about scope user.
type ScopeUser struct {
	baseObject
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
	byScope      *btreeIndex[int64, ScopeUser, *ScopeUser]
	byScopeLogin *btreeIndex[pair[int64, string], ScopeUser, *ScopeUser]
	salt         string
}

// FindByScope returns scope users by scope.
func (s *ScopeUserStore) FindByScope(scopeID ...int64) (db.Rows[ScopeUser], error) {
	s.mutex.RLock()
	return btreeIndexFind(
		s.byScope,
		s.objects.Iter(),
		s.mutex.RLocker(),
		scopeID,
		0,
	), nil
}

// GetByScopeLogin returns scope user by scope and login.
func (s *ScopeUserStore) GetByScopeLogin(scopeID int64, login string) (ScopeUser, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return btreeIndexGet(
		s.byScopeLogin,
		s.objects.Iter(),
		makePair(scopeID, strings.ToLower(login)),
	)
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
		byScope: newBTreeIndex(func(o ScopeUser) (int64, bool) { return o.ScopeID, true }, lessInt64),
		byScopeLogin: newBTreeIndex(
			func(o ScopeUser) (pair[int64, string], bool) {
				return makePair(o.ScopeID, strings.ToLower(o.Login)), true
			},
			lessPairInt64String,
		),
		salt: salt,
	}
	impl.cachedStore = makeCachedManualStore[ScopeUser, ScopeUserEvent](
		db, table, eventTable, impl, impl.byScope, impl.byScopeLogin,
	)
	return impl
}

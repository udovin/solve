package models

import (
	"crypto/rand"
	"database/sql"
	"strings"

	"github.com/udovin/gosql"
)

// InternalUser contains common information about internal user.
type InternalUser struct {
	baseObject
	AccountID    int64   `db:"account_id"`
	GroupID      int64   `db:"group_id"`
	Login        string  `db:"login"`
	PasswordHash string  `db:"password_hash"`
	PasswordSalt string  `db:"password_salt"`
	Title        NString `db:"title"`
}

// AccountKind returns InternalUserAccount kind.
func (o InternalUser) AccountKind() AccountKind {
	return InternalUserAccount
}

// Clone creates copy of internal user.
func (o InternalUser) Clone() InternalUser {
	return o
}

// InternalUserEvent represents an internal user event.
type InternalUserEvent struct {
	baseEvent
	InternalUser
}

// Object returns internal user.
func (e InternalUserEvent) Object() InternalUser {
	return e.InternalUser
}

// SetObject sets event internal user.
func (e *InternalUserEvent) SetObject(o InternalUser) {
	e.InternalUser = o
}

// UserStore represents users store.
type InternalUserStore struct {
	baseStore[InternalUser, InternalUserEvent, *InternalUser, *InternalUserEvent]
	byAccount    *index[int64, InternalUser, *InternalUser]
	byGroupLogin *index[pair[int64, string], InternalUser, *InternalUser]
	salt         string
}

var _ baseStoreImpl[InternalUser] = (*InternalUserStore)(nil)

// GetByLogin returns user by group and login.
func (s *InternalUserStore) GetByGroupLogin(group int64, login string) (InternalUser, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byGroupLogin.Get(makePair(group, strings.ToLower(login))) {
		if object, ok := s.objects[id]; ok {
			return object.Clone(), nil
		}
	}
	return InternalUser{}, sql.ErrNoRows
}

// GetByAccount returns internal user by account id.
func (s *InternalUserStore) GetByAccount(id int64) (InternalUser, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byAccount.Get(id) {
		if object, ok := s.objects[id]; ok {
			return object.Clone(), nil
		}
	}
	return InternalUser{}, sql.ErrNoRows
}

// SetPassword modifies PasswordHash and PasswordSalt fields.
//
// PasswordSalt will be replaced with random 16 byte string
// and PasswordHash will be calculated using password, salt
// and global salt.
func (s *InternalUserStore) SetPassword(user *InternalUser, password string) error {
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
func (s *InternalUserStore) CheckPassword(user InternalUser, password string) bool {
	passwordHash := hashPassword(password, user.PasswordSalt, s.salt)
	return passwordHash == user.PasswordHash
}

var _ baseStoreImpl[InternalUser] = (*InternalUserStore)(nil)

// NewInternalUserStore creates new instance of internal user store.
func NewInternalUserStore(
	db *gosql.DB, table, eventTable, salt string,
) *InternalUserStore {
	impl := &InternalUserStore{
		byAccount: newIndex(func(o InternalUser) int64 { return o.AccountID }),
		byGroupLogin: newIndex(func(o InternalUser) pair[int64, string] {
			return makePair(o.GroupID, strings.ToLower(o.Login))
		}),
		salt: salt,
	}
	impl.baseStore = makeBaseStore[InternalUser, InternalUserEvent](
		db, table, eventTable, impl, impl.byAccount, impl.byGroupLogin,
	)
	return impl
}

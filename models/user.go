package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/sha3"

	"github.com/udovin/gosql"
)

type UserStatus int

const (
	PendingUser UserStatus = 0
	ActiveUser  UserStatus = 1
	BlockedUser UserStatus = 2
)

// String returns string representation.
func (t UserStatus) String() string {
	switch t {
	case PendingUser:
		return "pending"
	case ActiveUser:
		return "active"
	case BlockedUser:
		return "blocked"
	default:
		return fmt.Sprintf("UserStatus(%d)", t)
	}
}

// User contains common information about user.
type User struct {
	baseObject
	AccountID    int64      `db:"account_id"`
	Login        string     `db:"login"`
	Status       UserStatus `db:"status"`
	PasswordHash string     `db:"password_hash"`
	PasswordSalt string     `db:"password_salt"`
	Email        NString    `db:"email"`
	FirstName    NString    `db:"first_name"`
	LastName     NString    `db:"last_name"`
	MiddleName   NString    `db:"middle_name"`
}

// AccountKind returns UserAccount kind.
func (o User) AccountKind() AccountKind {
	return UserAccount
}

// Clone creates copy of user.
func (o User) Clone() User {
	return o
}

// UserEvent represents an user event.
type UserEvent struct {
	baseEvent
	User
}

// Object returns user.
func (e UserEvent) Object() User {
	return e.User
}

// SetObject sets event user.
func (e *UserEvent) SetObject(o User) {
	e.User = o
}

// UserStore represents users store.
type UserStore struct {
	baseStore[User, UserEvent, *User, *UserEvent]
	byAccount *index[int64, User, *User]
	byLogin   *index[string, User, *User]
	salt      string
}

// GetByLogin returns user by login.
func (s *UserStore) GetByLogin(login string) (User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byLogin.Get(strings.ToLower(login)) {
		if object, ok := s.objects[id]; ok {
			return object.Clone(), nil
		}
	}
	return User{}, sql.ErrNoRows
}

// GetByAccount returns user by account id.
func (s *UserStore) GetByAccount(id int64) (User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byAccount.Get(id) {
		if object, ok := s.objects[id]; ok {
			return object.Clone(), nil
		}
	}
	return User{}, sql.ErrNoRows
}

// SetPassword modifies PasswordHash and PasswordSalt fields.
//
// PasswordSalt will be replaced with random 16 byte string
// and PasswordHash will be calculated using password, salt
// and global salt.
func (s *UserStore) SetPassword(user *User, password string) error {
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
func (s *UserStore) CheckPassword(user User, password string) bool {
	passwordHash := hashPassword(password, user.PasswordSalt, s.salt)
	return passwordHash == user.PasswordHash
}

var _ baseStoreImpl[User] = (*UserStore)(nil)

// NewUserStore creates new instance of user store.
func NewUserStore(
	db *gosql.DB, table, eventTable, salt string,
) *UserStore {
	impl := &UserStore{
		byAccount: newIndex(func(o User) int64 { return o.AccountID }),
		byLogin:   newIndex(func(o User) string { return strings.ToLower(o.Login) }),
		salt:      salt,
	}
	impl.baseStore = makeBaseStore[User, UserEvent](
		db, table, eventTable, impl, impl.byAccount, impl.byLogin,
	)
	return impl
}

func hashPassword(password, salt, globalSalt string) string {
	return hashString(salt + hashString(password) + globalSalt)
}

func encodeBase64(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

func hashString(value string) string {
	bytes := sha3.Sum512([]byte(value))
	return encodeBase64(bytes[:])
}

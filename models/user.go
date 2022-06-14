package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"strings"

	"golang.org/x/crypto/sha3"

	"github.com/udovin/gosql"
)

// User contains common information about user.
type User struct {
	ID           int64   `db:"id"`
	AccountID    int64   `db:"account_id"`
	Login        string  `db:"login"`
	PasswordHash string  `db:"password_hash"`
	PasswordSalt string  `db:"password_salt"`
	Email        NString `db:"email"`
	FirstName    NString `db:"first_name"`
	LastName     NString `db:"last_name"`
	MiddleName   NString `db:"middle_name"`
}

// AccountKind returns UserAccount kind.
func (o User) AccountKind() AccountKind {
	return UserAccount
}

// ObjectID returns ID of user.
func (o User) ObjectID() int64 {
	return o.ID
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
	baseStore[User, UserEvent]
	users     map[int64]User
	byAccount map[int64]int64
	byLogin   map[string]int64
	salt      string
}

// Get returns user by ID.
func (s *UserStore) Get(id int64) (User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if user, ok := s.users[id]; ok {
		return user.Clone(), nil
	}
	return User{}, sql.ErrNoRows
}

// GetByLogin returns user by login.
func (s *UserStore) GetByLogin(login string) (User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if id, ok := s.byLogin[strings.ToLower(login)]; ok {
		if user, ok := s.users[id]; ok {
			return user.Clone(), nil
		}
	}
	return User{}, sql.ErrNoRows
}

// GetByAccount returns user by login.
func (s *UserStore) GetByAccount(id int64) (User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if id, ok := s.byAccount[id]; ok {
		if user, ok := s.users[id]; ok {
			return user.Clone(), nil
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

func (s *UserStore) reset() {
	s.users = map[int64]User{}
	s.byAccount = map[int64]int64{}
	s.byLogin = map[string]int64{}
}

func (s *UserStore) makeObject(id int64) User {
	return User{ID: id}
}

func (s *UserStore) makeObjectEvent(typ EventType) UserEvent {
	return UserEvent{baseEvent: makeBaseEvent(typ)}
}

func (s *UserStore) onCreateObject(user User) {
	s.users[user.ID] = user
	s.byAccount[user.AccountID] = user.ID
	s.byLogin[strings.ToLower(user.Login)] = user.ID
}

func (s *UserStore) onDeleteObject(id int64) {
	if user, ok := s.users[id]; ok {
		delete(s.byAccount, user.AccountID)
		delete(s.byLogin, strings.ToLower(user.Login))
		delete(s.users, user.ID)
	}
}

// NewUserStore creates new instance of user store.
func NewUserStore(
	db *gosql.DB, table, eventTable, salt string,
) *UserStore {
	impl := &UserStore{salt: salt}
	impl.baseStore = makeBaseStore[User, UserEvent](
		db, table, eventTable, impl,
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

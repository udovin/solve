package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"

	"golang.org/x/crypto/sha3"

	"github.com/udovin/solve/db"
)

// User contains common information about user.
type User struct {
	ID           int64  `db:"id" json:""`
	AccountID    int64  `db:"account_id" json:"-"`
	Login        string `db:"login" json:""`
	PasswordHash string `db:"password_hash" json:"-"`
	PasswordSalt string `db:"password_salt" json:"-"`
}

// ObjectID returns ID of user.
func (o User) ObjectID() int64 {
	return o.ID
}

func (o User) clone() User {
	return o
}

// UserEvent represents an user event.
type UserEvent struct {
	baseEvent
	User
}

// Object returns user.
func (e UserEvent) Object() db.Object {
	return e.User
}

// WithObject return copy of event with replaced user.
func (e UserEvent) WithObject(o db.Object) ObjectEvent {
	e.User = o.(User)
	return e
}

// UserManager represents users manager.
type UserManager struct {
	baseManager
	users     map[int64]User
	byAccount map[int64]int64
	byLogin   map[string]int64
	salt      string
}

// Get returns user by ID.
func (m *UserManager) Get(id int64) (User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if user, ok := m.users[id]; ok {
		return user.clone(), nil
	}
	return User{}, sql.ErrNoRows
}

// GetByLogin returns user by login.
func (m *UserManager) GetByLogin(login string) (User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if id, ok := m.byLogin[login]; ok {
		if user, ok := m.users[id]; ok {
			return user.clone(), nil
		}
	}
	return User{}, sql.ErrNoRows
}

// GetByAccount returns user by login.
func (m *UserManager) GetByAccount(id int64) (User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if id, ok := m.byAccount[id]; ok {
		if user, ok := m.users[id]; ok {
			return user.clone(), nil
		}
	}
	return User{}, sql.ErrNoRows
}

// CreateTx creates user and returns copy with valid ID.
func (m *UserManager) CreateTx(tx *sql.Tx, user User) (User, error) {
	event, err := m.createObjectEvent(tx, UserEvent{
		makeBaseEvent(CreateEvent),
		user,
	})
	if err != nil {
		return User{}, err
	}
	return event.Object().(User), nil
}

// UpdateTx updates user with specified ID.
func (m *UserManager) UpdateTx(tx *sql.Tx, user User) error {
	_, err := m.createObjectEvent(tx, UserEvent{
		makeBaseEvent(UpdateEvent),
		user,
	})
	return err
}

// DeleteTx deletes user with specified ID.
func (m *UserManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, UserEvent{
		makeBaseEvent(DeleteEvent),
		User{ID: id},
	})
	return err
}

// SetPassword modifies PasswordHash and PasswordSalt fields.
//
// PasswordSalt will be replaced with random 16 byte string
// and PasswordHash will be calculated using password, salt
// and global salt.
func (m *UserManager) SetPassword(user *User, password string) error {
	saltBytes := make([]byte, 16)
	_, err := rand.Read(saltBytes)
	if err != nil {
		return err
	}
	user.PasswordSalt = encodeBase64(saltBytes)
	user.PasswordHash = hashPassword(password, user.PasswordSalt, m.salt)
	return nil
}

// CheckPassword checks that passwords are the same.
func (m *UserManager) CheckPassword(user User, password string) bool {
	passwordHash := hashPassword(password, user.PasswordSalt, m.salt)
	return passwordHash == user.PasswordHash
}

func (m *UserManager) reset() {
	m.users = map[int64]User{}
	m.byAccount = map[int64]int64{}
	m.byLogin = map[string]int64{}
}

func (m *UserManager) onCreateObject(o db.Object) {
	user := o.(User)
	m.users[user.ID] = user
	m.byAccount[user.AccountID] = user.ID
	m.byLogin[user.Login] = user.ID
}

func (m *UserManager) onDeleteObject(o db.Object) {
	user := o.(User)
	delete(m.byAccount, user.AccountID)
	delete(m.byLogin, user.Login)
	delete(m.users, user.ID)
}

func (m *UserManager) onUpdateObject(o db.Object) {
	user := o.(User)
	if old, ok := m.users[user.ID]; ok {
		if old.AccountID != user.AccountID {
			delete(m.byAccount, old.AccountID)
		}
		if old.Login != user.Login {
			delete(m.byLogin, old.Login)
		}
	}
	m.onCreateObject(o)
}

// NewUserManager creates new instance of user manager.
func NewUserManager(table, eventTable, salt string, dialect db.Dialect) *UserManager {
	impl := &UserManager{salt: salt}
	impl.baseManager = makeBaseManager(
		User{}, table, UserEvent{}, eventTable, impl, dialect,
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

package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/sha3"
)

// User contains common information about user
type User struct {
	ID           int64  `json:""  db:"id"`
	Login        string `json:""  db:"login"`
	PasswordHash string `json:"-" db:"password_hash"`
	PasswordSalt string `json:"-" db:"password_salt"`
	CreateTime   int64  `json:""  db:"create_time"`
	IsSuper      bool   `json:""  db:"is_super"`
}

type UserChange struct {
	BaseChange
	User
}

// UserStore represents cached store for users
type UserStore struct {
	Manager     *ChangeManager
	table       string
	changeTable string
	users       map[int64]User
	loginUsers  map[string]int64
	mutex       sync.RWMutex
}

// SetPassword modifies PasswordHash and PasswordSalt fields
//
// PasswordSalt will be replaced with random 16 byte string and
// PasswordHash will be calculated using password, salt and PasswordSalt.
func (m *User) SetPassword(password, salt string) error {
	saltBytes := make([]byte, 16)
	_, err := rand.Read(saltBytes)
	if err != nil {
		return err
	}
	m.PasswordSalt = encodeBase64(saltBytes)
	m.PasswordHash = m.hashPassword(password, salt)
	return nil
}

// CheckPassword checks that passwords are the same
func (m *User) CheckPassword(password, salt string) bool {
	passwordHash := m.hashPassword(password, salt)
	return passwordHash == m.PasswordHash
}

// NewUserStore creates new instance of user store
func NewUserStore(db *sql.DB, table, changeTable string) *UserStore {
	store := UserStore{
		table:       table,
		changeTable: changeTable,
		users:       make(map[int64]User),
		loginUsers:  make(map[string]int64),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

// Get returns user by ID
func (s *UserStore) Get(id int64) (User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if user, ok := s.users[id]; ok {
		return user, nil
	}
	return User{}, sql.ErrNoRows
}

// GetByLogin returns user by login
func (s *UserStore) GetByLogin(login string) (User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if id, ok := s.loginUsers[login]; ok {
		if user, ok := s.users[id]; ok {
			return user, nil
		}
	}
	return User{}, sql.ErrNoRows
}

// Create creates new user
func (s *UserStore) Create(m *User) error {
	change := UserChange{
		BaseChange: BaseChange{Type: CreateChange},
		User:       *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.User
	return nil
}

// CreateTx creates new user
func (s *UserStore) CreateTx(tx *ChangeTx, m *User) error {
	change := UserChange{
		BaseChange: BaseChange{Type: CreateChange},
		User:       *m,
	}
	err := s.Manager.ChangeTx(tx, &change)
	if err != nil {
		return err
	}
	*m = change.User
	return nil
}

// Update modifies user data
func (s *UserStore) Update(m *User) error {
	change := UserChange{
		BaseChange: BaseChange{Type: UpdateChange},
		User:       *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.User
	return nil
}

// Delete deletes user with specified id
func (s *UserStore) Delete(id int64) error {
	change := UserChange{
		BaseChange: BaseChange{Type: DeleteChange},
		User:       User{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *UserStore) GetLocker() sync.Locker {
	return &s.mutex
}

func (s *UserStore) InitChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *UserStore) LoadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time", "id",`+
				` "login", "password_hash", "password_salt", "create_time",`+
				` "is_super"`+
				` FROM %q`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *UserStore) ScanChange(scan Scanner) (Change, error) {
	user := UserChange{}
	err := scan.Scan(
		&user.BaseChange.ID, &user.Type, &user.Time,
		&user.User.ID, &user.Login, &user.PasswordHash,
		&user.PasswordSalt, &user.CreateTime, &user.IsSuper,
	)
	return &user, err
}

func (s *UserStore) SaveChange(tx *sql.Tx, change Change) error {
	user := change.(*UserChange)
	user.Time = time.Now().Unix()
	switch user.Type {
	case CreateChange:
		user.CreateTime = user.Time
		var err error
		user.User.ID, err = execTxReturningID(
			s.Manager.db.Driver(), tx,
			fmt.Sprintf(
				`INSERT INTO %q`+
					` ("login", "password_hash", "password_salt",`+
					` "create_time", "is_super")`+
					` VALUES ($1, $2, $3, $4, $5)`,
				s.table,
			),
			"id",
			user.Login, user.PasswordHash, user.PasswordSalt,
			user.CreateTime, user.IsSuper,
		)
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.users[user.User.ID]; !ok {
			return fmt.Errorf(
				"user with id = %d does not exists",
				user.User.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE %q SET`+
					` "login" = $1, "password_hash" = $2,`+
					` "password_salt" = $3, "create_time" = $4,`+
					` "is_super" = $5`+
					` WHERE "id" = $6`,
				s.table,
			),
			user.Login, user.PasswordHash, user.PasswordSalt,
			user.CreateTime, user.IsSuper, user.User.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.users[user.User.ID]; !ok {
			return fmt.Errorf(
				"user with id = %d does not exists",
				user.User.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM %q WHERE "id" = $1`,
				s.table,
			),
			user.User.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			user.Type,
		)
	}
	var err error
	user.BaseChange.ID, err = execTxReturningID(
		s.Manager.db.Driver(), tx,
		fmt.Sprintf(
			`INSERT INTO %q`+
				` ("change_type", "change_time",`+
				` "id", "login", "password_hash", "password_salt",`+
				` "create_time", "is_super")`+
				` VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			s.changeTable,
		),
		"change_id",
		user.Type, user.Time, user.User.ID, user.Login, user.PasswordHash,
		user.PasswordSalt, user.CreateTime, user.IsSuper,
	)
	return err
}

func (s *UserStore) ApplyChange(change Change) {
	user := change.(*UserChange)
	switch user.Type {
	case UpdateChange:
		if old, ok := s.users[user.User.ID]; ok {
			if old.Login != user.Login {
				delete(s.loginUsers, old.Login)
			}
		}
		fallthrough
	case CreateChange:
		s.loginUsers[user.Login] = user.User.ID
		s.users[user.User.ID] = user.User
	case DeleteChange:
		delete(s.loginUsers, user.Login)
		delete(s.users, user.User.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			user.Type,
		))
	}
}

func (m *User) hashPassword(password, salt string) string {
	return hashString(m.PasswordSalt + hashString(password) + salt)
}

func encodeBase64(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

func hashString(value string) string {
	bytes := sha3.Sum512([]byte(value))
	return encodeBase64(bytes[:])
}

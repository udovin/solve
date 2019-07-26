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

type User struct {
	ID           int64  `json:""  db:"id"`
	Login        string `json:""  db:"login"`
	PasswordHash string `json:"-" db:"password_hash"`
	PasswordSalt string `json:"-" db:"password_salt"`
	CreateTime   int64  `json:""  db:"create_time"`
}

type userChange struct {
	BaseChange
	User
}

type UserStore struct {
	Manager     *ChangeManager
	table       string
	changeTable string
	users       map[int64]User
	loginUsers  map[string]int64
	mutex       sync.RWMutex
}

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

func (m *User) CheckPassword(password, salt string) bool {
	passwordHash := m.hashPassword(password, salt)
	return passwordHash == m.PasswordHash
}

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

func (s *UserStore) Get(id int64) (User, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	user, ok := s.users[id]
	return user, ok
}

func (s *UserStore) GetByLogin(login string) (User, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	id, ok := s.loginUsers[login]
	if !ok {
		return User{}, ok
	}
	return s.Get(id)
}

func (s *UserStore) Create(m *User) error {
	change := userChange{
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

func (s *UserStore) Update(m *User) error {
	change := userChange{
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

func (s *UserStore) Delete(id int64) error {
	change := userChange{
		BaseChange: BaseChange{Type: DeleteChange},
		User:       User{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *UserStore) getLocker() sync.Locker {
	return &s.mutex
}

func (s *UserStore) setupChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *UserStore) loadChangeGapTx(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time", "id",`+
				` "login", "password_hash", "password_salt", "create_time"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *UserStore) scanChange(scan Scanner) (Change, error) {
	user := userChange{}
	err := scan.Scan(
		&user.BaseChange.ID, &user.Type, &user.Time,
		&user.User.ID, &user.Login, &user.PasswordHash,
		&user.PasswordSalt, &user.CreateTime,
	)
	return &user, err
}

func (s *UserStore) saveChangeTx(tx *sql.Tx, change Change) error {
	user := change.(*userChange)
	user.Time = time.Now().Unix()
	switch user.Type {
	case CreateChange:
		user.CreateTime = user.Time
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("login", "password_hash",`+
					` "password_salt", "create_time")`+
					` VALUES ($1, $2, $3, $4)`,
				s.table,
			),
			user.Login, user.PasswordHash,
			user.PasswordSalt, user.CreateTime,
		)
		if err != nil {
			return err
		}
		user.User.ID, err = res.LastInsertId()
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
				`UPDATE "%s" SET`+
					` "login" = $1, "password_hash" = $2,`+
					` "password_salt" = $3`+
					` WHERE "id" = $4`,
				s.table,
			),
			user.Login, user.PasswordHash,
			user.PasswordSalt, user.User.ID,
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
				`DELETE FROM "%s" WHERE "id" = $1`,
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
	res, err := tx.Exec(
		fmt.Sprintf(
			`INSERT INTO "%s" `+
				`("change_type", "change_time", `+
				`"id", "login", "password_hash", `+
				`"password_salt", "create_time") `+
				`VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			s.changeTable,
		),
		user.Type, user.Time, user.User.ID, user.Login,
		user.PasswordHash, user.PasswordSalt, user.CreateTime,
	)
	if err != nil {
		return err
	}
	user.BaseChange.ID, err = res.LastInsertId()
	return err
}

func (s *UserStore) applyChange(change Change) {
	user := change.(*userChange)
	switch user.Type {
	case UpdateChange:
		if oldUser, ok := s.users[user.User.ID]; ok {
			if oldUser.Login != user.Login {
				delete(s.loginUsers, oldUser.Login)
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

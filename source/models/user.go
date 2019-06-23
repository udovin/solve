package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/sha3"
)

type User struct {
	ID           int64  `db:"id"            json:""`
	Login        string `db:"login"         json:""`
	PasswordHash string `db:"password_hash" json:"-"`
	PasswordSalt string `db:"password_salt" json:"-"`
	CreateTime   int64  `db:"create_time"   json:""`
}

type UserChange struct {
	User
	ChangeBase
}

type UserStore struct {
	Manager     *ChangeManager
	db          *sql.DB
	table       string
	changeTable string
	users       map[int64]User
	loginMap    map[string]int64
}

func (c *UserChange) ChangeData() interface{} {
	return c.User
}

func NewUserStore(
	db *sql.DB, table, changeTable string,
) *UserStore {
	store := UserStore{
		db: db, table: table, changeTable: changeTable,
		users:    make(map[int64]User),
		loginMap: make(map[string]int64),
	}
	store.Manager = NewChangeManager(&store)
	return &store
}

func (s *UserStore) GetDB() *sql.DB {
	return s.db
}

func (s *UserStore) TableName() string {
	return s.table
}

func (s *UserStore) ChangeTableName() string {
	return s.changeTable
}

func (s *UserStore) Get(id int64) (User, bool) {
	user, ok := s.users[id]
	return user, ok
}

func (s *UserStore) GetByLogin(login string) (User, bool) {
	id, ok := s.loginMap[login]
	if !ok {
		return User{}, ok
	}
	return s.Get(id)
}

func (s *UserStore) Create(m *User) error {
	change := UserChange{
		ChangeBase: ChangeBase{Type: CreateChange},
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
	change := UserChange{
		ChangeBase: ChangeBase{Type: UpdateChange},
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
	change := UserChange{
		ChangeBase: ChangeBase{Type: DeleteChange},
		User:       User{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *UserStore) scanChange(scan RowScan) (Change, error) {
	change := &UserChange{}
	err := scan.Scan(
		&change.ChangeBase.ID, &change.Type, &change.Time,
		&change.User.ID, &change.Login,
		&change.PasswordHash, &change.PasswordSalt,
		&change.CreateTime,
	)
	if err != nil {
		return nil, err
	}
	return change, nil
}

func (s *UserStore) saveChangeTx(tx *sql.Tx, change Change) error {
	user := change.(*UserChange)
	user.Time = time.Now().Unix()
	switch user.Type {
	case CreateChange:
		user.CreateTime = user.Time
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s" `+
					`("login", "password_hash", `+
					`"password_salt", "create_time") `+
					`VALUES ($1, $2, $3, $4)`,
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
				`UPDATE "%s" SET `+
					`"login" = $1, "password_hash" = $2, `+
					`"password_salt" = $3 `+
					`WHERE "id" = $4`,
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
			s.ChangeTableName(),
		),
		user.Type, user.Time, user.User.ID, user.Login,
		user.PasswordHash, user.PasswordSalt, user.CreateTime,
	)
	if err != nil {
		return err
	}
	user.ChangeBase.ID, err = res.LastInsertId()
	return err
}

func (s *UserStore) applyChange(change Change) {
	userChange := change.(*UserChange)
	user := userChange.User
	switch userChange.Type {
	case CreateChange:
		s.loginMap[user.Login] = user.ID
		s.users[user.ID] = user
	case UpdateChange:
		if oldUser, ok := s.users[user.ID]; ok {
			if oldUser.Login != user.Login {
				delete(s.loginMap, oldUser.Login)
			}
		}
		s.loginMap[user.Login] = user.ID
		s.users[user.ID] = user
	case DeleteChange:
		delete(s.loginMap, user.Login)
		delete(s.users, user.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			userChange.Type,
		))
	}
}

func encodeBase64(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

func hashString(value string) string {
	bytes := sha3.Sum512([]byte(value))
	return encodeBase64(bytes[:])
}

func (m *User) hashPassword(password, salt string) string {
	return hashString(m.PasswordSalt + hashString(password) + salt)
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

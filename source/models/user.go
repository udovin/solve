package models

import (
	"database/sql"
	"fmt"
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
	ID   int64      `db:"change_id"   json:""`
	Type ChangeType `db:"change_type" json:""`
	Time int64      `db:"change_time" json:""`
}

type UserStore struct {
	Manager     *ChangeManager
	db          *sql.DB
	table       string
	changeTable string
	users       map[int64]User
}

func (c *UserChange) ChangeID() int64 {
	return c.ID
}

func (c *UserChange) ChangeType() ChangeType {
	return c.Type
}

func (c *UserChange) ChangeTime() int64 {
	return c.Time
}

func (c *UserChange) ChangeData() interface{} {
	return c.User
}

func NewUserStore(
	db *sql.DB, table, changeTable string,
) *UserStore {
	store := UserStore{
		db: db, table: table, changeTable: changeTable,
		users: make(map[int64]User),
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

func (s *UserStore) scanChange(scan RowScan) (Change, error) {
	change := &UserChange{}
	err := scan.Scan(
		&change.ID, &change.Type, &change.Time,
		&change.User.ID, &change.Login,
		&change.PasswordHash, &change.PasswordSalt,
		&change.CreateTime,
	)
	if err != nil {
		return nil, err
	}
	return change, nil
}

func (s *UserStore) createChangeTx(
	tx *sql.Tx, changeType ChangeType, changeTime int64, data interface{},
) (Change, error) {
	var user User
	switch changeType {
	case CreateChange:
		user = data.(User)
		user.CreateTime = changeTime
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
			return nil, err
		}
		sessionID, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		user.ID = sessionID
	case UpdateChange:
		user = data.(User)
		if _, ok := s.users[user.ID]; !ok {
			return nil, fmt.Errorf(
				"user with id = %d does not exists", user.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET `+
					`'"login" = $2, "password_hash" = $3, `+
					`"password_salt" = $4 `+
					`WHERE "id" = $1"`,
				s.table,
			),
			user.ID, user.Login, user.PasswordHash, user.PasswordSalt,
		)
		if err != nil {
			return nil, err
		}
	case DeleteChange:
		var ok bool
		user, ok = s.users[data.(int64)]
		if !ok {
			return nil, fmt.Errorf(
				"user with id = %d does not exists", user.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1"`,
				s.table,
			),
			user.ID,
		)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf(
			"unsupported change type = %d", changeType,
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
		changeType, changeTime, user.ID, user.Login,
		user.PasswordHash, user.PasswordSalt, user.CreateTime,
	)
	if err != nil {
		return nil, err
	}
	changeID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &UserChange{
		ID: changeID, Type: changeType,
		Time: changeTime, User: user,
	}, nil
}

func (s *UserStore) applyChange(change Change) {
	user := change.ChangeData().(User)
	switch change.ChangeType() {
	case CreateChange:
		s.users[user.ID] = user
	case UpdateChange:
		s.users[user.ID] = user
	case DeleteChange:
		delete(s.users, user.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %d", change.ChangeType(),
		))
	}
}

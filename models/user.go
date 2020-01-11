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
	Login        string `db:"login" json:""`
	PasswordHash string `db:"password_hash" json:"-"`
	PasswordSalt string `db:"password_salt" json:"-"`
	CreateTime   int64  `db:"create_time" json:""`
	IsSuper      bool   `db:"is_super" json:""`
}

// ObjectID returns ID of user.
func (o User) ObjectID() int64 {
	return o.ID
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
	users   map[int64]User
	byLogin map[string]int64
	salt    string
}

// Get returns user by ID.
func (m *UserManager) Get(id int64) (User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return User{}, sql.ErrNoRows
}

// GetByLogin returns user by login.
func (m *UserManager) GetByLogin(login string) (User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if id, ok := m.byLogin[login]; ok {
		if user, ok := m.users[id]; ok {
			return user, nil
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
	m.byLogin = map[string]int64{}
}

func (m *UserManager) onCreateObject(o db.Object) {
	user := o.(User)
	m.users[user.ID] = user
	m.byLogin[user.Login] = user.ID
}

func (m *UserManager) onDeleteObject(o db.Object) {
	user := o.(User)
	delete(m.byLogin, user.Login)
	delete(m.users, user.ID)
}

func (m *UserManager) onUpdateObject(o db.Object) {
	user := o.(User)
	if old, ok := m.users[user.ID]; ok {
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

/*
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
*/
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

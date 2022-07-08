package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

// ContestUser contains common information about contest user.
type ContestUser struct {
	ID           int64  `db:"id"`
	AccountID    int64  `db:"account_id"`
	ContestID    int64  `db:"contest_id"`
	Login        string `db:"login"`
	PasswordHash string `db:"password_hash"`
	PasswordSalt string `db:"password_salt"`
	Name         string `db:"name"`
}

// ObjectID returns ID of user.
func (o ContestUser) ObjectID() int64 {
	return o.ID
}

// SetObjectID sets ID of contest user.
func (o *ContestUser) SetObjectID(id int64) {
	o.ID = id
}

// Clone creates copy of user.
func (o ContestUser) Clone() ContestUser {
	return o
}

// ContestUserEvent represents an contest user event.
type ContestUserEvent struct {
	baseEvent
	ContestUser
}

// Object returns contest user.
func (e ContestUserEvent) Object() ContestUser {
	return e.ContestUser
}

// SetObject sets event contest user.
func (e *ContestUserEvent) SetObject(o ContestUser) {
	e.ContestUser = o
}

// UserStore represents users store.
type ContestUserStore struct {
	baseStore[ContestUser, ContestUserEvent]
	users map[int64]ContestUser
}

// Get returns user by ID.
func (s *ContestUserStore) Get(id int64) (ContestUser, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if user, ok := s.users[id]; ok {
		return user.Clone(), nil
	}
	return ContestUser{}, sql.ErrNoRows
}

func (s *ContestUserStore) reset() {
	s.users = map[int64]ContestUser{}
}

func (s *ContestUserStore) makeObject(id int64) ContestUser {
	return ContestUser{ID: id}
}

func (s *ContestUserStore) makeObjectEvent(typ EventType) ContestUserEvent {
	return ContestUserEvent{baseEvent: makeBaseEvent(typ)}
}

func (s *ContestUserStore) onCreateObject(user ContestUser) {
	s.users[user.ID] = user
}

func (s *ContestUserStore) onDeleteObject(id int64) {
	if user, ok := s.users[id]; ok {
		delete(s.users, user.ID)
	}
}

// NewContestUserStore creates new instance of contest user store.
func NewContestUserStore(
	db *gosql.DB, table, eventTable, salt string,
) *ContestUserStore {
	impl := &ContestUserStore{}
	impl.baseStore = makeBaseStore[ContestUser, ContestUserEvent](
		db, table, eventTable, impl,
	)
	return impl
}

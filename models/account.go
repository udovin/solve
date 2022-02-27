package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

// AccountKind represents kind of account.
type AccountKind int

const (
	// UserAccount represents kind of account for user.
	UserAccount AccountKind = 1
)

// Account represents an account.
type Account struct {
	// ID contains ID of account.
	ID int64 `db:"id"`
	// Kind contains kind of account.
	Kind AccountKind `db:"kind"`
}

// ObjectID return ID of account.
func (o Account) ObjectID() int64 {
	return o.ID
}

// Clone creates copy of account.
func (o Account) Clone() Account {
	return o
}

// AccountEvent represents an account event.
type AccountEvent struct {
	baseEvent
	Account
}

// Object returns event account.
func (e AccountEvent) Object() Account {
	return e.Account
}

// WithObject returns event with replaced Account.
func (e AccountEvent) WithObject(o Account) ObjectEvent[Account] {
	e.Account = o
	return e
}

// AccountStore represents store for accounts.
type AccountStore struct {
	baseStore[Account, AccountEvent]
	accounts map[int64]Account
}

// Get returns account by ID.
func (s *AccountStore) Get(id int64) (Account, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if account, ok := s.accounts[id]; ok {
		return account.Clone(), nil
	}
	return Account{}, sql.ErrNoRows
}

func (s *AccountStore) reset() {
	s.accounts = map[int64]Account{}
}

func (s *AccountStore) makeObject(id int64) Account {
	return Account{ID: id}
}

func (s *AccountStore) makeObjectEvent(typ EventType) ObjectEvent[Account] {
	return AccountEvent{baseEvent: makeBaseEvent(typ)}
}

func (s *AccountStore) onCreateObject(account Account) {
	s.accounts[account.ID] = account
}

func (s *AccountStore) onDeleteObject(account Account) {
	delete(s.accounts, account.ID)
}

func (s *AccountStore) onUpdateObject(account Account) {
	if old, ok := s.accounts[account.ID]; ok {
		s.onDeleteObject(old)
	}
	s.onCreateObject(account)
}

// NewAccountStore creates a new instance of AccountStore.
func NewAccountStore(
	db *gosql.DB, table, eventTable string,
) *AccountStore {
	impl := &AccountStore{}
	impl.baseStore = makeBaseStore[Account, AccountEvent](
		db, table, eventTable, impl,
	)
	return impl
}

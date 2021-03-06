package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
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
func (e AccountEvent) Object() db.Object {
	return e.Account
}

// WithObject returns event with replaced Account.
func (e AccountEvent) WithObject(o db.Object) ObjectEvent {
	e.Account = o.(Account)
	return e
}

// AccountStore represents store for accounts.
type AccountStore struct {
	baseStore
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

// CreateTx creates account and returns copy with valid ID.
func (s *AccountStore) CreateTx(
	tx *sql.Tx, account Account,
) (Account, error) {
	event, err := s.createObjectEvent(tx, AccountEvent{
		makeBaseEvent(CreateEvent),
		account,
	})
	if err != nil {
		return Account{}, err
	}
	return event.Object().(Account), nil
}

// UpdateTx updates account with specified ID.
func (s *AccountStore) UpdateTx(tx *sql.Tx, account Account) error {
	_, err := s.createObjectEvent(tx, AccountEvent{
		makeBaseEvent(UpdateEvent),
		account,
	})
	return err
}

// DeleteTx deletes account with specified ID.
func (s *AccountStore) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := s.createObjectEvent(tx, AccountEvent{
		makeBaseEvent(DeleteEvent),
		Account{ID: id},
	})
	return err
}

func (s *AccountStore) reset() {
	s.accounts = map[int64]Account{}
}

func (s *AccountStore) onCreateObject(o db.Object) {
	account := o.(Account)
	s.accounts[account.ID] = account
}

func (s *AccountStore) onDeleteObject(o db.Object) {
	account := o.(Account)
	delete(s.accounts, account.ID)
}

func (s *AccountStore) onUpdateObject(o db.Object) {
	s.onCreateObject(o)
}

// NewAccountStore creates a new instance of AccountStore.
func NewAccountStore(
	table, eventTable string, dialect db.Dialect,
) *AccountStore {
	impl := &AccountStore{}
	impl.baseStore = makeBaseStore(
		Account{}, table, AccountEvent{}, eventTable, impl, dialect,
	)
	return impl
}

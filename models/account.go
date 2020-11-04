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

func (o Account) clone() Account {
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

// AccountManager represents manager for accounts.
type AccountManager struct {
	baseManager
	accounts map[int64]Account
}

// Get returns account by ID.
func (m *AccountManager) Get(id int64) (Account, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if account, ok := m.accounts[id]; ok {
		return account.clone(), nil
	}
	return Account{}, sql.ErrNoRows
}

// CreateTx creates account and returns copy with valid ID.
func (m *AccountManager) CreateTx(
	tx *sql.Tx, account Account,
) (Account, error) {
	event, err := m.createObjectEvent(tx, AccountEvent{
		makeBaseEvent(CreateEvent),
		account,
	})
	if err != nil {
		return Account{}, err
	}
	return event.Object().(Account), nil
}

// UpdateTx updates account with specified ID.
func (m *AccountManager) UpdateTx(tx *sql.Tx, account Account) error {
	_, err := m.createObjectEvent(tx, AccountEvent{
		makeBaseEvent(UpdateEvent),
		account,
	})
	return err
}

// DeleteTx deletes account with specified ID.
func (m *AccountManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, AccountEvent{
		makeBaseEvent(DeleteEvent),
		Account{ID: id},
	})
	return err
}

func (m *AccountManager) reset() {
	m.accounts = map[int64]Account{}
}

func (m *AccountManager) onCreateObject(o db.Object) {
	account := o.(Account)
	m.accounts[account.ID] = account
}

func (m *AccountManager) onDeleteObject(o db.Object) {
	account := o.(Account)
	delete(m.accounts, account.ID)
}

func (m *AccountManager) onUpdateObject(o db.Object) {
	m.onCreateObject(o)
}

// NewAccountManager creates a new instance of AccountManager.
func NewAccountManager(
	table, eventTable string, dialect db.Dialect,
) *AccountManager {
	impl := &AccountManager{}
	impl.baseManager = makeBaseManager(
		Account{}, table, AccountEvent{}, eventTable, impl, dialect,
	)
	return impl
}

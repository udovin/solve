package models

import (
	"github.com/udovin/gosql"
)

// AccountKind represents kind of account.
type AccountKind int

const (
	// UserAccount represents kind of account for user.
	UserAccount AccountKind = 1
	// InternalUserAccount represents kind of account for internal user.
	InternalUserAccount AccountKind = 2
)

// Account represents an account.
type Account struct {
	baseObject
	// Kind contains kind of account.
	Kind AccountKind `db:"kind"`
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

// SetObject sets event account.
func (e *AccountEvent) SetObject(o Account) {
	e.Account = o
}

// AccountStore represents store for accounts.
type AccountStore struct {
	baseStore[Account, AccountEvent, *Account, *AccountEvent]
}

var _ baseStoreImpl[Account] = (*AccountStore)(nil)

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

package models

import (
	"fmt"

	"github.com/udovin/gosql"
)

// AccountKind represents kind of account.
type AccountKind int

const (
	// UserAccount represents kind of account for user.
	UserAccountKind AccountKind = 1
	// ScopeUserAccount represents kind of account for scope user.
	ScopeUserAccountKind AccountKind = 2
	// ScopeAccount represents kind of account for scope.
	ScopeAccountKind AccountKind = 3
	// GroupAccount represents kind of account for group.
	GroupAccountKind AccountKind = 4
)

// String returns string representation.
func (k AccountKind) String() string {
	switch k {
	case UserAccountKind:
		return "user"
	case ScopeUserAccountKind:
		return "scope_user"
	case ScopeAccountKind:
		return "scope"
	case GroupAccountKind:
		return "group"
	default:
		return fmt.Sprintf("AccountKind(%d)", k)
	}
}

func (k AccountKind) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}

func (k *AccountKind) UnmarshalText(data []byte) error {
	switch s := string(data); s {
	case "user":
		*k = UserAccountKind
	case "scope_user":
		*k = ScopeUserAccountKind
	case "scope":
		*k = ScopeAccountKind
	case "group":
		*k = GroupAccountKind
	default:
		return fmt.Errorf("unsupported kind: %q", s)
	}
	return nil
}

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
	cachedStore[Account, AccountEvent, *Account, *AccountEvent]
}

// NewAccountStore creates a new instance of AccountStore.
func NewAccountStore(
	db *gosql.DB, table, eventTable string,
) *AccountStore {
	impl := &AccountStore{}
	impl.cachedStore = makeCachedStore[Account, AccountEvent](
		db, table, eventTable, impl,
	)
	return impl
}

package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

// Scope represents a scope for users.
type Scope struct {
	baseObject
	AccountID int64  `db:"account_id"`
	OwnerID   NInt64 `db:"owner_id"`
	Title     string `db:"title"`
}

// AccountKind returns ScopeAccount kind.
func (o Scope) AccountKind() AccountKind {
	return ScopeAccount
}

// Clone creates copy of scope.
func (o Scope) Clone() Scope {
	return o
}

// ScopeEvent represents an scope event.
type ScopeEvent struct {
	baseEvent
	Scope
}

// Object returns event scope.
func (e ScopeEvent) Object() Scope {
	return e.Scope
}

// SetObject sets event scope.
func (e *ScopeEvent) SetObject(o Scope) {
	e.Scope = o
}

// ScopeStore represents store for scopes.
type ScopeStore struct {
	cachedStore[Scope, ScopeEvent, *Scope, *ScopeEvent]
	byAccount *index[int64, Scope, *Scope]
}

// GetByAccount returns scope user by account id.
func (s *ScopeStore) GetByAccount(id int64) (Scope, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byAccount.Get(id) {
		if object, ok := s.objects.Get(id); ok {
			return object.Clone(), nil
		}
	}
	return Scope{}, sql.ErrNoRows
}

// NewScopeStore creates a new instance of ScopeStore.
func NewScopeStore(
	db *gosql.DB, table, eventTable string,
) *ScopeStore {
	impl := &ScopeStore{
		byAccount: newIndex(func(o Scope) int64 { return o.AccountID }),
	}
	impl.cachedStore = makeCachedStore[Scope, ScopeEvent](
		db, table, eventTable, impl, impl.byAccount,
	)
	return impl
}

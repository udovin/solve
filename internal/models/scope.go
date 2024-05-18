package models

import (
	"github.com/udovin/gosql"
)

// Scope represents a scope for users.
type Scope struct {
	baseObject
	OwnerID NInt64 `db:"owner_id"`
	Title   string `db:"title"`
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
}

// NewScopeStore creates a new instance of ScopeStore.
func NewScopeStore(
	db *gosql.DB, table, eventTable string,
) *ScopeStore {
	impl := &ScopeStore{}
	impl.cachedStore = makeCachedManualStore[Scope, ScopeEvent](
		db, table, eventTable, impl,
	)
	return impl
}

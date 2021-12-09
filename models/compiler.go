package models

import (
	"github.com/udovin/gosql"
	"github.com/udovin/solve/db"
)

// Compiler represents compiler.
type Compiler struct {
	ID int64 `db:"id"`
}

// ObjectID returns ID of compiler.
func (o Compiler) ObjectID() int64 {
	return o.ID
}

// Clone creates copy of compiler.
func (o Compiler) Clone() Compiler {
	return o
}

// CompilerEvent represents compiler event.
type CompilerEvent struct {
	baseEvent
	Compiler
}

// Object returns event compiler.
func (e CompilerEvent) Object() db.Object {
	return e.Compiler
}

// WithObject replaces event compiler.
func (e CompilerEvent) WithObject(o db.Object) ObjectEvent {
	e.Compiler = o.(Compiler)
	return e
}

// CompilerStore represents store for compilers.
type CompilerStore struct {
	baseStore
	compilers map[int64]Compiler
}

func (s *CompilerStore) reset() {
	s.compilers = map[int64]Compiler{}
}

func (s *CompilerStore) onCreateObject(o db.Object) {
	compiler := o.(Compiler)
	s.compilers[compiler.ID] = compiler
}

func (s *CompilerStore) onDeleteObject(o db.Object) {
	compiler := o.(Compiler)
	delete(s.compilers, compiler.ID)
}

func (s *CompilerStore) onUpdateObject(o db.Object) {
	compiler := o.(Compiler)
	if old, ok := s.compilers[compiler.ID]; ok {
		s.onDeleteObject(old)
	}
	s.onCreateObject(o)
}

// NewCompilerStore creates a new instance of CompilerStore.
func NewCompilerStore(db *gosql.DB, table, eventTable string) *CompilerStore {
	impl := &CompilerStore{}
	impl.baseStore = makeBaseStore(
		Compiler{}, table, CompilerEvent{}, eventTable, impl, db.Dialect(),
	)
	return impl
}

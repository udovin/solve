package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

// Compiler represents compiler.
type Compiler struct {
	ID      int64  `db:"id"`
	OwnerID NInt64 `db:"owner_id"`
	Name    string `db:"name"`
	Config  JSON   `db:"config"`
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
func (e CompilerEvent) Object() Compiler {
	return e.Compiler
}

// WithObject replaces event compiler.
func (e CompilerEvent) WithObject(o Compiler) ObjectEvent[Compiler] {
	e.Compiler = o
	return e
}

// CompilerStore represents store for compilers.
type CompilerStore struct {
	baseStore[Compiler, CompilerEvent]
	compilers map[int64]Compiler
}

// Get returns compiler by specified ID.
func (s *CompilerStore) Get(id int64) (Compiler, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if compiler, ok := s.compilers[id]; ok {
		return compiler.Clone(), nil
	}
	return Compiler{}, sql.ErrNoRows
}

func (s *CompilerStore) All() ([]Compiler, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var compilers []Compiler
	for _, compiler := range s.compilers {
		compilers = append(compilers, compiler)
	}
	return compilers, nil
}

func (s *CompilerStore) reset() {
	s.compilers = map[int64]Compiler{}
}

func (s *CompilerStore) makeObject(id int64) Compiler {
	return Compiler{ID: id}
}

func (s *CompilerStore) makeObjectEvent(typ EventType) ObjectEvent[Compiler] {
	return CompilerEvent{baseEvent: makeBaseEvent(typ)}
}

func (s *CompilerStore) onCreateObject(compiler Compiler) {
	s.compilers[compiler.ID] = compiler
}

func (s *CompilerStore) onDeleteObject(compiler Compiler) {
	delete(s.compilers, compiler.ID)
}

func (s *CompilerStore) onUpdateObject(compiler Compiler) {
	if old, ok := s.compilers[compiler.ID]; ok {
		s.onDeleteObject(old)
	}
	s.onCreateObject(compiler)
}

// NewCompilerStore creates a new instance of CompilerStore.
func NewCompilerStore(db *gosql.DB, table, eventTable string) *CompilerStore {
	impl := &CompilerStore{}
	impl.baseStore = makeBaseStore[Compiler, CompilerEvent](
		db, table, eventTable, impl,
	)
	return impl
}

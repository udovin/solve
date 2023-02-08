package models

import (
	"database/sql"
	"encoding/json"

	"github.com/udovin/gosql"
)

type CompilerCommandConfig struct {
	Command string   `json:"command"`
	Environ []string `json:"environ"`
	Workdir string   `json:"workdir"`
	Source  *string  `json:"source,omitempty"`
	Binary  *string  `json:"binary,omitempty"`
}

type CompilerConfig struct {
	Language   string                 `json:"language,omitempty"`
	Compiler   string                 `json:"compiler,omitempty"`
	Extensions []string               `json:"extensions"`
	Compile    *CompilerCommandConfig `json:"compile,omitempty"`
	Execute    *CompilerCommandConfig `json:"execute,omitempty"`
}

// Compiler represents compiler.
type Compiler struct {
	baseObject
	OwnerID NInt64 `db:"owner_id"`
	Name    string `db:"name"`
	Config  JSON   `db:"config"`
	ImageID int64  `db:"image_id"`
}

// Clone create copy of compiler.
func (o Compiler) Clone() Compiler {
	o.Config = o.Config.Clone()
	return o
}

func (o Compiler) GetConfig() (CompilerConfig, error) {
	var config CompilerConfig
	if len(o.Config) == 0 {
		return config, nil
	}
	err := json.Unmarshal(o.Config, &config)
	return config, err
}

func (o *Compiler) SetConfig(config CompilerConfig) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	o.Config = raw
	return nil
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

// SetObject sets event compiler.
func (e *CompilerEvent) SetObject(o Compiler) {
	e.Compiler = o
}

// CompilerStore represents store for compilers.
type CompilerStore struct {
	cachedStore[Compiler, CompilerEvent, *Compiler, *CompilerEvent]
	byName *index[string, Compiler, *Compiler]
}

// GetByName returns compiler by name.
func (s *CompilerStore) GetByName(name string) (Compiler, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byName.Get(name) {
		if object, ok := s.objects.Get(id); ok {
			return object.Clone(), nil
		}
	}
	return Compiler{}, sql.ErrNoRows
}

var _ baseStoreImpl[Compiler] = (*CompilerStore)(nil)

// NewCompilerStore creates a new instance of CompilerStore.
func NewCompilerStore(db *gosql.DB, table, eventTable string) *CompilerStore {
	impl := &CompilerStore{
		byName: newIndex(func(o Compiler) string { return o.Name }),
	}
	impl.cachedStore = makeBaseStore[Compiler, CompilerEvent](
		db, table, eventTable, impl, impl.byName,
	)
	return impl
}

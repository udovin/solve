package models

import (
	"encoding/json"

	"github.com/udovin/gosql"
)

type ProblemConfig struct {
	TimeLimit   int64 `json:"time_limit,omitempty"`
	MemoryLimit int64 `json:"memory_limit,omitempty"`
}

// Problem represents a problem.
type Problem struct {
	baseObject
	OwnerID    NInt64 `db:"owner_id"`
	Config     JSON   `db:"config"`
	Title      string `db:"title"`
	PackageID  NInt64 `db:"package_id"`
	CompiledID NInt64 `db:"compiled_id"`
}

func (o Problem) GetConfig() (ProblemConfig, error) {
	var config ProblemConfig
	if len(o.Config) == 0 {
		return config, nil
	}
	err := json.Unmarshal(o.Config, &config)
	return config, err
}

func (o *Problem) SetConfig(config ProblemConfig) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	o.Config = raw
	return nil
}

// Clone creates copy of problem.
func (o Problem) Clone() Problem {
	o.Config = o.Config.Clone()
	return o
}

// ProblemEvent represents a problem event.
type ProblemEvent struct {
	baseEvent
	Problem
}

// Object returns event problem.
func (e ProblemEvent) Object() Problem {
	return e.Problem
}

// SetObject sets event problem.
func (e *ProblemEvent) SetObject(o Problem) {
	e.Problem = o
}

// ProblemStore represents store for problems.
type ProblemStore struct {
	cachedStore[Problem, ProblemEvent, *Problem, *ProblemEvent]
}

// NewProblemStore creates a new instance of ProblemStore.
func NewProblemStore(
	db *gosql.DB, table, eventTable string,
) *ProblemStore {
	impl := &ProblemStore{}
	impl.cachedStore = makeCachedStore[Problem, ProblemEvent](
		db, table, eventTable, impl,
	)
	return impl
}

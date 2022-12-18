package models

import (
	"database/sql"
	"encoding/json"

	"github.com/udovin/gosql"
)

type ProblemConfig struct {
	TimeLimit   int64 `json:"time_limit"`
	MemoryLimit int64 `json:"memory_limit"`
}

// Problem represents a problem.
type Problem struct {
	baseObject
	OwnerID   NInt64 `db:"owner_id"`
	Config    JSON   `db:"config"`
	Title     string `db:"title"`
	PackageID NInt64 `db:"package_id"`
}

func (o Problem) GetConfig() (ProblemConfig, error) {
	var config ProblemConfig
	if len(o.Config) == 0 {
		return config, nil
	}
	err := json.Unmarshal(o.Config, &config)
	return config, err
}

func (o *Problem) SetMeta(config ProblemConfig) error {
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
	baseStore[Problem, ProblemEvent, *Problem, *ProblemEvent]
	problems map[int64]Problem
}

// Get returns problem by ID.
//
// If there is no problem with specified ID then
// sql.ErrNoRows will be returned.
func (s *ProblemStore) Get(id int64) (Problem, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if problem, ok := s.problems[id]; ok {
		return problem.Clone(), nil
	}
	return Problem{}, sql.ErrNoRows
}

// All returns all problems.
func (s *ProblemStore) All() ([]Problem, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var problems []Problem
	for _, problem := range s.problems {
		problems = append(problems, problem)
	}
	return problems, nil
}

//lint:ignore U1000 Used in generic interface.
func (s *ProblemStore) reset() {
	s.problems = map[int64]Problem{}
}

//lint:ignore U1000 Used in generic interface.
func (s *ProblemStore) onCreateObject(problem Problem) {
	s.problems[problem.ID] = problem
}

//lint:ignore U1000 Used in generic interface.
func (s *ProblemStore) onDeleteObject(id int64) {
	if problem, ok := s.problems[id]; ok {
		delete(s.problems, problem.ID)
	}
}

var _ baseStoreImpl[Problem] = (*ProblemStore)(nil)

// NewProblemStore creates a new instance of ProblemStore.
func NewProblemStore(
	db *gosql.DB, table, eventTable string,
) *ProblemStore {
	impl := &ProblemStore{}
	impl.baseStore = makeBaseStore[Problem, ProblemEvent](
		db, table, eventTable, impl,
	)
	return impl
}

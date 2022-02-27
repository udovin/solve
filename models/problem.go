package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

// Problem represents a problem.
type Problem struct {
	ID      int64  `db:"id"`
	OwnerID NInt64 `db:"owner_id"`
	Config  JSON   `db:"config"`
	Title   string `db:"title"`
}

// ObjectID return ID of problem.
func (o Problem) ObjectID() int64 {
	return o.ID
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

// WithObject returns event with replaced Problem.
func (e ProblemEvent) WithObject(o Problem) ObjectEvent[Problem] {
	e.Problem = o
	return e
}

// ProblemStore represents store for problems.
type ProblemStore struct {
	baseStore[Problem, ProblemEvent]
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

func (s *ProblemStore) reset() {
	s.problems = map[int64]Problem{}
}

func (s *ProblemStore) makeObject(id int64) Problem {
	return Problem{ID: id}
}

func (s *ProblemStore) makeObjectEvent(typ EventType) ObjectEvent[Problem] {
	return ProblemEvent{baseEvent: makeBaseEvent(typ)}
}

func (s *ProblemStore) onCreateObject(problem Problem) {
	s.problems[problem.ID] = problem
}

func (s *ProblemStore) onDeleteObject(problem Problem) {
	delete(s.problems, problem.ID)
}

func (s *ProblemStore) onUpdateObject(problem Problem) {
	if old, ok := s.problems[problem.ID]; ok {
		s.onDeleteObject(old)
	}
	s.onCreateObject(problem)
}

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

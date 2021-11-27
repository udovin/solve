package models

import (
	"database/sql"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/db"
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
func (e ProblemEvent) Object() db.Object {
	return e.Problem
}

// WithObject returns event with replaced Problem.
func (e ProblemEvent) WithObject(o db.Object) ObjectEvent {
	e.Problem = o.(Problem)
	return e
}

// ProblemStore represents store for problems.
type ProblemStore struct {
	baseStore
	problems map[int64]Problem
}

// CreateTx creates problem and returns copy with valid ID.
func (s *ProblemStore) CreateTx(tx gosql.WeakTx, problem *Problem) error {
	event, err := s.createObjectEvent(tx, ProblemEvent{
		makeBaseEvent(CreateEvent), *problem,
	})
	if err != nil {
		return err
	}
	*problem = event.Object().(Problem)
	return nil
}

// UpdateTx updates problem with specified ID.
func (s *ProblemStore) UpdateTx(tx gosql.WeakTx, problem Problem) error {
	_, err := s.createObjectEvent(tx, ProblemEvent{
		makeBaseEvent(UpdateEvent),
		problem,
	})
	return err
}

// DeleteTx deletes problem with specified ID.
func (s *ProblemStore) DeleteTx(tx gosql.WeakTx, id int64) error {
	_, err := s.createObjectEvent(tx, ProblemEvent{
		makeBaseEvent(DeleteEvent),
		Problem{ID: id},
	})
	return err
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

func (s *ProblemStore) onCreateObject(o db.Object) {
	problem := o.(Problem)
	s.problems[problem.ID] = problem
}

func (s *ProblemStore) onDeleteObject(o db.Object) {
	problem := o.(Problem)
	delete(s.problems, problem.ID)
}

func (s *ProblemStore) onUpdateObject(o db.Object) {
	s.onCreateObject(o)
}

// NewProblemStore creates a new instance of ProblemStore.
func NewProblemStore(
	table, eventTable string, dialect gosql.Dialect,
) *ProblemStore {
	impl := &ProblemStore{}
	impl.baseStore = makeBaseStore(
		Problem{}, table, ProblemEvent{}, eventTable, impl, dialect,
	)
	return impl
}

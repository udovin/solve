package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
)

// Problem represents a problem.
type Problem struct {
	ID     int64 `db:"id"`
	Config JSON  `db:"config"`
}

// ObjectID return ID of problem.
func (o Problem) ObjectID() int64 {
	return o.ID
}

// Clone creates copy of problem.
func (o Problem) Clone() Problem {
	o.Config = o.Config.clone()
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
func (s *ProblemStore) CreateTx(
	tx *sql.Tx, problem Problem,
) (Problem, error) {
	event, err := s.createObjectEvent(tx, ProblemEvent{
		makeBaseEvent(CreateEvent),
		problem,
	})
	if err != nil {
		return Problem{}, err
	}
	return event.Object().(Problem), nil
}

// UpdateTx updates problem with specified ID.
func (s *ProblemStore) UpdateTx(tx *sql.Tx, problem Problem) error {
	_, err := s.createObjectEvent(tx, ProblemEvent{
		makeBaseEvent(UpdateEvent),
		problem,
	})
	return err
}

// DeleteTx deletes problem with specified ID.
func (s *ProblemStore) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := s.createObjectEvent(tx, ProblemEvent{
		makeBaseEvent(DeleteEvent),
		Problem{ID: id},
	})
	return err
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
	table, eventTable string, dialect db.Dialect,
) *ProblemStore {
	impl := &ProblemStore{}
	impl.baseStore = makeBaseStore(
		Problem{}, table, ProblemEvent{}, eventTable, impl, dialect,
	)
	return impl
}

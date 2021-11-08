package models

import (
	"database/sql"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/db"
)

// Solution represents a solution.
type Solution struct {
	ID        int64 `db:"id"`
	ProblemID int64 `db:"problem_id"`
	AuthorID  int64 `db:"author_id"`
}

// ObjectID return ID of solution.
func (o Solution) ObjectID() int64 {
	return o.ID
}

// Clone creates copy of solution.
func (o Solution) Clone() Solution {
	return o
}

// SolutionEvent represents a solution event.
type SolutionEvent struct {
	baseEvent
	Solution
}

// Object returns event solution.
func (e SolutionEvent) Object() db.Object {
	return e.Solution
}

// WithObject returns event with replaced Solution.
func (e SolutionEvent) WithObject(o db.Object) ObjectEvent {
	e.Solution = o.(Solution)
	return e
}

// SolutionStore represents store for solutions.
type SolutionStore struct {
	baseStore
	solutions map[int64]Solution
}

// CreateTx creates solution and returns copy with valid ID.
func (s *SolutionStore) CreateTx(
	tx *sql.Tx, solution Solution,
) (Solution, error) {
	event, err := s.createObjectEvent(tx, SolutionEvent{
		makeBaseEvent(CreateEvent),
		solution,
	})
	if err != nil {
		return Solution{}, err
	}
	return event.Object().(Solution), nil
}

// UpdateTx updates solution with specified ID.
func (s *SolutionStore) UpdateTx(tx *sql.Tx, solution Solution) error {
	_, err := s.createObjectEvent(tx, SolutionEvent{
		makeBaseEvent(UpdateEvent),
		solution,
	})
	return err
}

// DeleteTx deletes solution with specified ID.
func (s *SolutionStore) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := s.createObjectEvent(tx, SolutionEvent{
		makeBaseEvent(DeleteEvent),
		Solution{ID: id},
	})
	return err
}

// Get returns solution by ID.
//
// If there is no solution with specified ID then
// sql.ErrNoRows will be returned.
func (s *SolutionStore) Get(id int64) (Solution, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if solution, ok := s.solutions[id]; ok {
		return solution.Clone(), nil
	}
	return Solution{}, sql.ErrNoRows
}

// All returns all solutions.
func (s *SolutionStore) All() ([]Solution, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var solutions []Solution
	for _, solution := range s.solutions {
		solutions = append(solutions, solution)
	}
	return solutions, nil
}

func (s *SolutionStore) reset() {
	s.solutions = map[int64]Solution{}
}

func (s *SolutionStore) onCreateObject(o db.Object) {
	solution := o.(Solution)
	s.solutions[solution.ID] = solution
}

func (s *SolutionStore) onDeleteObject(o db.Object) {
	solution := o.(Solution)
	delete(s.solutions, solution.ID)
}

func (s *SolutionStore) onUpdateObject(o db.Object) {
	s.onCreateObject(o)
}

// NewSolutionStore creates a new instance of SolutionStore.
func NewSolutionStore(
	table, eventTable string, dialect gosql.Dialect,
) *SolutionStore {
	impl := &SolutionStore{}
	impl.baseStore = makeBaseStore(
		Solution{}, table, SolutionEvent{}, eventTable, impl, dialect,
	)
	return impl
}

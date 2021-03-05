package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
)

// ContestProblem represents connection for problems.
type ContestProblem struct {
	// ID contains ID of role.
	ID int64 `db:"id"`
	// ContestID contains ID of contest.
	ContestID int64 `db:"contest_id"`
	// ProblemID contains ID of problem.
	ProblemID int64 `db:"problem_id"`
	// Code contains code of problem.
	Code string `db:"code"`
}

// ObjectID return ID of problem.
func (o ContestProblem) ObjectID() int64 {
	return o.ID
}

// Clone creates copy of contest problem.
func (o ContestProblem) Clone() ContestProblem {
	return o
}

// ContestProblemEvent represents problem event.
type ContestProblemEvent struct {
	baseEvent
	ContestProblem
}

// Object returns event role edge.
func (e ContestProblemEvent) Object() db.Object {
	return e.ContestProblem
}

// WithObject returns event with replaced ContestProblem.
func (e ContestProblemEvent) WithObject(o db.Object) ObjectEvent {
	e.ContestProblem = o.(ContestProblem)
	return e
}

// ContestProblemStore represents a problem store.
type ContestProblemStore struct {
	baseStore
	problems  map[int64]ContestProblem
	byContest indexInt64
}

// Get returns problem by ID.
//
// If there is no problem with specified ID then
// sql.ErrNoRows will be returned.
func (s *ContestProblemStore) Get(id int64) (ContestProblem, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if problem, ok := s.problems[id]; ok {
		return problem.Clone(), nil
	}
	return ContestProblem{}, sql.ErrNoRows
}

// FindByContest returns problems by parent ID.
func (s *ContestProblemStore) FindByContest(
	contestID int64,
) ([]ContestProblem, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var problems []ContestProblem
	for id := range s.byContest[contestID] {
		if problem, ok := s.problems[id]; ok {
			problems = append(problems, problem.Clone())
		}
	}
	return problems, nil
}

// CreateTx creates problem and returns copy with valid ID.
func (s *ContestProblemStore) CreateTx(
	tx *sql.Tx, problem ContestProblem,
) (ContestProblem, error) {
	event, err := s.createObjectEvent(tx, ContestProblemEvent{
		makeBaseEvent(CreateEvent),
		problem,
	})
	if err != nil {
		return ContestProblem{}, err
	}
	return event.Object().(ContestProblem), nil
}

// UpdateTx updates problem with specified ID.
func (s *ContestProblemStore) UpdateTx(
	tx *sql.Tx, problem ContestProblem,
) error {
	_, err := s.createObjectEvent(tx, ContestProblemEvent{
		makeBaseEvent(UpdateEvent),
		problem,
	})
	return err
}

// DeleteTx deletes problem with specified ID.
func (s *ContestProblemStore) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := s.createObjectEvent(tx, ContestProblemEvent{
		makeBaseEvent(DeleteEvent),
		ContestProblem{ID: id},
	})
	return err
}

func (s *ContestProblemStore) reset() {
	s.problems = map[int64]ContestProblem{}
	s.byContest = indexInt64{}
}

func (s *ContestProblemStore) onCreateObject(o db.Object) {
	problem := o.(ContestProblem)
	s.problems[problem.ID] = problem
	s.byContest.Create(problem.ContestID, problem.ID)
}

func (s *ContestProblemStore) onDeleteObject(o db.Object) {
	problem := o.(ContestProblem)
	s.byContest.Delete(problem.ContestID, problem.ID)
	delete(s.problems, problem.ID)
}

func (s *ContestProblemStore) onUpdateObject(o db.Object) {
	problem := o.(ContestProblem)
	if old, ok := s.problems[problem.ID]; ok {
		if old.ContestID != problem.ContestID {
			s.byContest.Delete(old.ContestID, old.ID)
		}
	}
	s.onCreateObject(o)
}

// NewContestProblemStore creates a new instance of ContestProblemStore.
func NewContestProblemStore(
	table, eventTable string, dialect db.Dialect,
) *ContestProblemStore {
	impl := &ContestProblemStore{}
	impl.baseStore = makeBaseStore(
		ContestProblem{}, table, ContestProblemEvent{}, eventTable, impl, dialect,
	)
	return impl
}

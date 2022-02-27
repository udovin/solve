package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

// ContestProblem represents connection for problems.
type ContestProblem struct {
	// ID contains ID of role.
	ID int64 `db:"id"`
	// ProblemID contains ID of problem.
	ProblemID int64 `db:"problem_id"`
	// ContestID contains ID of contest.
	ContestID int64 `db:"contest_id"`
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
func (e ContestProblemEvent) Object() ContestProblem {
	return e.ContestProblem
}

// WithObject returns event with replaced ContestProblem.
func (e ContestProblemEvent) WithObject(o ContestProblem) ObjectEvent[ContestProblem] {
	e.ContestProblem = o
	return e
}

// ContestProblemStore represents a problem store.
type ContestProblemStore struct {
	baseStore[ContestProblem, ContestProblemEvent]
	problems  map[int64]ContestProblem
	byContest index[int64]
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

// DeleteTx deletes problem with specified ID.
func (s *ContestProblemStore) DeleteTx(tx gosql.WeakTx, id int64) error {
	_, err := s.createObjectEvent(tx, ContestProblemEvent{
		makeBaseEvent(DeleteEvent),
		ContestProblem{ID: id},
	})
	return err
}

func (s *ContestProblemStore) reset() {
	s.problems = map[int64]ContestProblem{}
	s.byContest = makeIndex[int64]()
}

func (s *ContestProblemStore) makeObjectEvent(typ EventType) ObjectEvent[ContestProblem] {
	return ContestProblemEvent{baseEvent: makeBaseEvent(typ)}
}

func (s *ContestProblemStore) onCreateObject(problem ContestProblem) {
	s.problems[problem.ID] = problem
	s.byContest.Create(problem.ContestID, problem.ID)
}

func (s *ContestProblemStore) onDeleteObject(problem ContestProblem) {
	s.byContest.Delete(problem.ContestID, problem.ID)
	delete(s.problems, problem.ID)
}

func (s *ContestProblemStore) onUpdateObject(problem ContestProblem) {
	if old, ok := s.problems[problem.ID]; ok {
		s.onDeleteObject(old)
	}
	s.onCreateObject(problem)
}

// NewContestProblemStore creates a new instance of ContestProblemStore.
func NewContestProblemStore(
	db *gosql.DB, table, eventTable string,
) *ContestProblemStore {
	impl := &ContestProblemStore{}
	impl.baseStore = makeBaseStore[ContestProblem, ContestProblemEvent](
		db, table, eventTable, impl,
	)
	return impl
}

package models

import (
	"github.com/udovin/gosql"
)

// ContestProblem represents connection for problems.
type ContestProblem struct {
	baseObject
	// ProblemID contains ID of problem.
	ProblemID int64 `db:"problem_id"`
	// ContestID contains ID of contest.
	ContestID int64 `db:"contest_id"`
	// Code contains code of problem.
	Code string `db:"code"`
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

// SetObject sets event contest problem.
func (e *ContestProblemEvent) SetObject(o ContestProblem) {
	e.ContestProblem = o
}

// ContestProblemStore represents a problem store.
type ContestProblemStore struct {
	baseStore[ContestProblem, ContestProblemEvent, *ContestProblem, *ContestProblemEvent]
	byContest *index[int64, ContestProblem, *ContestProblem]
}

// FindByContest returns problems by parent ID.
func (s *ContestProblemStore) FindByContest(id int64) ([]ContestProblem, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []ContestProblem
	for id := range s.byContest.Get(id) {
		if object, ok := s.objects[id]; ok {
			objects = append(objects, object.Clone())
		}
	}
	return objects, nil
}

var _ baseStoreImpl[ContestProblem] = (*ContestProblemStore)(nil)

// NewContestProblemStore creates a new instance of ContestProblemStore.
func NewContestProblemStore(
	db *gosql.DB, table, eventTable string,
) *ContestProblemStore {
	impl := &ContestProblemStore{
		byContest: newIndex(func(o ContestProblem) int64 { return o.ContestID }),
	}
	impl.baseStore = makeBaseStore[ContestProblem, ContestProblemEvent](
		db, table, eventTable, impl, impl.byContest,
	)
	return impl
}

package models

import (
	"github.com/udovin/gosql"
)

// ContestSolution represents connection for solutions.
type ContestSolution struct {
	baseObject
	// SolutionID contains ID of solution.
	SolutionID int64 `db:"solution_id"`
	// ContestID contains ID of contest.
	ContestID int64 `db:"contest_id"`
	// ParticipantID contains ID of participant.
	ParticipantID int64 `db:"participant_id"`
	// ProblemID contains ID of contest problem.
	ProblemID int64 `db:"problem_id"`
}

// Clone creates copy of contest solution.
func (o ContestSolution) Clone() ContestSolution {
	return o
}

// ContestSolutionEvent represents solution event.
type ContestSolutionEvent struct {
	baseEvent
	ContestSolution
}

// Object returns event role edge.
func (e ContestSolutionEvent) Object() ContestSolution {
	return e.ContestSolution
}

// SetObject sets event contest solution.
func (e *ContestSolutionEvent) SetObject(o ContestSolution) {
	e.ContestSolution = o
}

// ContestSolutionStore represents a solution store.
type ContestSolutionStore struct {
	cachedStore[ContestSolution, ContestSolutionEvent, *ContestSolution, *ContestSolutionEvent]
	byContest     *index[int64, ContestSolution, *ContestSolution]
	byParticipant *index[int64, ContestSolution, *ContestSolution]
}

// FindByContest returns solutions by contest ID.
func (s *ContestSolutionStore) FindByContest(
	id int64,
) ([]ContestSolution, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []ContestSolution
	for id := range s.byContest.Get(id) {
		if object, ok := s.objects[id]; ok {
			objects = append(objects, object.Clone())
		}
	}
	return objects, nil
}

// FindByContest returns solutions by participant ID.
func (s *ContestSolutionStore) FindByParticipant(
	id int64,
) ([]ContestSolution, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []ContestSolution
	for id := range s.byParticipant.Get(id) {
		if object, ok := s.objects[id]; ok {
			objects = append(objects, object.Clone())
		}
	}
	return objects, nil
}

var _ baseStoreImpl[ContestSolution] = (*ContestSolutionStore)(nil)

// NewContestSolutionStore creates a new instance of ContestSolutionStore.
func NewContestSolutionStore(
	db *gosql.DB, table, eventTable string,
) *ContestSolutionStore {
	impl := &ContestSolutionStore{
		byContest:     newIndex(func(o ContestSolution) int64 { return o.ContestID }),
		byParticipant: newIndex(func(o ContestSolution) int64 { return o.ParticipantID }),
	}
	impl.cachedStore = makeBaseStore[ContestSolution, ContestSolutionEvent](
		db, table, eventTable, impl, impl.byContest, impl.byParticipant,
	)
	return impl
}

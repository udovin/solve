package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

// ContestSolution represents connection for solutions.
type ContestSolution struct {
	// ID contains ID of role.
	ID int64 `db:"id"`
	// SolutionID contains ID of solution.
	SolutionID int64 `db:"solution_id"`
	// ContestID contains ID of contest.
	ContestID int64 `db:"contest_id"`
	// ParticipantID contains ID of participant.
	ParticipantID int64 `db:"participant_id"`
	// ProblemID contains ID of contest problem.
	ProblemID int64 `db:"problem_id"`
}

// ObjectID return ID of contest solution.
func (o ContestSolution) ObjectID() int64 {
	return o.ID
}

// SetObjectID sets ID of contest solution.
func (o *ContestSolution) SetObjectID(id int64) {
	o.ID = id
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
	baseStore[ContestSolution, ContestSolutionEvent, *ContestSolution, *ContestSolutionEvent]
	solutions     map[int64]ContestSolution
	byContest     index[int64]
	byParticipant index[int64]
}

// Get returns solution by ID.
//
// If there is no solution with specified ID then
// sql.ErrNoRows will be returned.
func (s *ContestSolutionStore) Get(id int64) (ContestSolution, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if solution, ok := s.solutions[id]; ok {
		return solution.Clone(), nil
	}
	return ContestSolution{}, sql.ErrNoRows
}

// FindByContest returns solutions by parent ID.
func (s *ContestSolutionStore) FindByContest(
	contestID int64,
) ([]ContestSolution, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var solutions []ContestSolution
	for id := range s.byContest[contestID] {
		if solution, ok := s.solutions[id]; ok {
			solutions = append(solutions, solution.Clone())
		}
	}
	return solutions, nil
}

func (s *ContestSolutionStore) reset() {
	s.solutions = map[int64]ContestSolution{}
	s.byContest = makeIndex[int64]()
	s.byParticipant = makeIndex[int64]()
}

func (s *ContestSolutionStore) makeObject(id int64) ContestSolution {
	return ContestSolution{ID: id}
}

func (s *ContestSolutionStore) makeObjectEvent(typ EventType) ContestSolutionEvent {
	return ContestSolutionEvent{baseEvent: makeBaseEvent(typ)}
}

func (s *ContestSolutionStore) onCreateObject(solution ContestSolution) {
	s.solutions[solution.ID] = solution
	s.byContest.Create(solution.ContestID, solution.ID)
	s.byParticipant.Create(solution.ParticipantID, solution.ID)
}

func (s *ContestSolutionStore) onDeleteObject(id int64) {
	if solution, ok := s.solutions[id]; ok {
		s.byContest.Delete(solution.ContestID, solution.ID)
		s.byParticipant.Delete(solution.ParticipantID, solution.ID)
		delete(s.solutions, solution.ID)
	}
}

// NewContestSolutionStore creates a new instance of ContestSolutionStore.
func NewContestSolutionStore(
	db *gosql.DB, table, eventTable string,
) *ContestSolutionStore {
	impl := &ContestSolutionStore{}
	impl.baseStore = makeBaseStore[ContestSolution, ContestSolutionEvent, *ContestSolution, *ContestSolutionEvent](
		db, table, eventTable, impl,
	)
	return impl
}

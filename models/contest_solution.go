package models

import (
	"database/sql"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/db"
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

// ObjectID return ID of solution.
func (o ContestSolution) ObjectID() int64 {
	return o.ID
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
func (e ContestSolutionEvent) Object() db.Object {
	return e.ContestSolution
}

// WithObject returns event with replaced ContestSolution.
func (e ContestSolutionEvent) WithObject(o db.Object) ObjectEvent {
	e.ContestSolution = o.(ContestSolution)
	return e
}

// ContestSolutionStore represents a solution store.
type ContestSolutionStore struct {
	baseStore
	solutions     map[int64]ContestSolution
	byContest     indexInt64
	byParticipant indexInt64
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

// CreateTx creates solution and returns copy with valid ID.
func (s *ContestSolutionStore) CreateTx(
	tx gosql.WeakTx, solution ContestSolution,
) (ContestSolution, error) {
	event, err := s.createObjectEvent(tx, ContestSolutionEvent{
		makeBaseEvent(CreateEvent),
		solution,
	})
	if err != nil {
		return ContestSolution{}, err
	}
	return event.Object().(ContestSolution), nil
}

// UpdateTx updates solution with specified ID.
func (s *ContestSolutionStore) UpdateTx(
	tx gosql.WeakTx, solution ContestSolution,
) error {
	_, err := s.createObjectEvent(tx, ContestSolutionEvent{
		makeBaseEvent(UpdateEvent),
		solution,
	})
	return err
}

// DeleteTx deletes solution with specified ID.
func (s *ContestSolutionStore) DeleteTx(tx gosql.WeakTx, id int64) error {
	_, err := s.createObjectEvent(tx, ContestSolutionEvent{
		makeBaseEvent(DeleteEvent),
		ContestSolution{ID: id},
	})
	return err
}

func (s *ContestSolutionStore) reset() {
	s.solutions = map[int64]ContestSolution{}
	s.byContest = indexInt64{}
	s.byParticipant = indexInt64{}
}

func (s *ContestSolutionStore) onCreateObject(o db.Object) {
	solution := o.(ContestSolution)
	s.solutions[solution.ID] = solution
	s.byContest.Create(solution.ContestID, solution.ID)
	s.byParticipant.Create(solution.ParticipantID, solution.ID)
}

func (s *ContestSolutionStore) onDeleteObject(o db.Object) {
	solution := o.(ContestSolution)
	s.byContest.Delete(solution.ContestID, solution.ID)
	s.byParticipant.Delete(solution.ParticipantID, solution.ID)
	delete(s.solutions, solution.ID)
}

func (s *ContestSolutionStore) onUpdateObject(o db.Object) {
	solution := o.(ContestSolution)
	if old, ok := s.solutions[solution.ID]; ok {
		s.onDeleteObject(old)
	}
	s.onCreateObject(o)
}

// NewContestSolutionStore creates a new instance of ContestSolutionStore.
func NewContestSolutionStore(
	table, eventTable string, dialect gosql.Dialect,
) *ContestSolutionStore {
	impl := &ContestSolutionStore{}
	impl.baseStore = makeBaseStore(
		ContestSolution{}, table, ContestSolutionEvent{}, eventTable, impl, dialect,
	)
	return impl
}

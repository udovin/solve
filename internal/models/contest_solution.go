package models

import (
	"context"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
)

// ContestSolution represents connection for solutions.
type ContestSolution struct {
	baseObject
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
	byContest     *btreeIndex[int64, ContestSolution, *ContestSolution]
	byParticipant *btreeIndex[int64, ContestSolution, *ContestSolution]
}

// FindByContest returns solutions by contest ID.
func (s *ContestSolutionStore) FindByContest(
	ctx context.Context, contestID ...int64,
) (db.Rows[ContestSolution], error) {
	s.mutex.RLock()
	return btreeIndexFind(
		s.byContest,
		s.objects.Iter(),
		s.mutex.RLocker(),
		contestID,
		0,
	), nil
}

func (s *ContestSolutionStore) ReverseFindByContestFrom(
	ctx context.Context,
	contestID []int64,
	beginID int64,
) (db.Rows[ContestSolution], error) {
	s.mutex.RLock()
	return btreeIndexReverseFind(
		s.byContest,
		s.objects.Iter(),
		s.mutex.RLocker(),
		contestID,
		beginID,
	), nil
}

// FindByContest returns solutions by participant ID.
func (s *ContestSolutionStore) FindByParticipant(
	ctx context.Context, participantID ...int64,
) (db.Rows[ContestSolution], error) {
	s.mutex.RLock()
	return btreeIndexFind(
		s.byParticipant,
		s.objects.Iter(),
		s.mutex.RLocker(),
		participantID,
		0,
	), nil
}

func (s *ContestSolutionStore) ReverseFindByParticipantFrom(
	ctx context.Context,
	participantID []int64,
	beginID int64,
) (db.Rows[ContestSolution], error) {
	s.mutex.RLock()
	return btreeIndexReverseFind(
		s.byParticipant,
		s.objects.Iter(),
		s.mutex.RLocker(),
		participantID,
		beginID,
	), nil
}

// NewContestSolutionStore creates a new instance of ContestSolutionStore.
func NewContestSolutionStore(
	db *gosql.DB, table, eventTable string,
) *ContestSolutionStore {
	impl := &ContestSolutionStore{
		byContest:     newBTreeIndex(func(o ContestSolution) (int64, bool) { return o.ContestID, true }, lessInt64),
		byParticipant: newBTreeIndex(func(o ContestSolution) (int64, bool) { return o.ParticipantID, true }, lessInt64),
	}
	impl.cachedStore = makeCachedManualStore[ContestSolution, ContestSolutionEvent](
		db, table, eventTable, impl, impl.byContest, impl.byParticipant,
	)
	return impl
}

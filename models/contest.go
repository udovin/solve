package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

// Contest represents a contest.
type Contest struct {
	ID      int64  `db:"id"`
	OwnerID NInt64 `db:"owner_id"`
	Config  JSON   `db:"config"`
	Title   string `db:"title"`
}

// ObjectID return ID of contest.
func (o Contest) ObjectID() int64 {
	return o.ID
}

// Clone creates copy of contest.
func (o Contest) Clone() Contest {
	o.Config = o.Config.Clone()
	return o
}

// ContestEvent represents a contest event.
type ContestEvent struct {
	baseEvent
	Contest
}

// Object returns event contest.
func (e ContestEvent) Object() Contest {
	return e.Contest
}

// WithObject returns event with replaced Contest.
func (e ContestEvent) WithObject(o Contest) ObjectEvent[Contest] {
	e.Contest = o
	return e
}

// ContestStore represents store for contests.
type ContestStore struct {
	baseStore[Contest, ContestEvent]
	contests map[int64]Contest
}

// Get returns contest by ID.
//
// If there is no contest with specified ID then
// sql.ErrNoRows will be returned.
func (s *ContestStore) Get(id int64) (Contest, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if contest, ok := s.contests[id]; ok {
		return contest.Clone(), nil
	}
	return Contest{}, sql.ErrNoRows
}

// All returns all contests.
func (s *ContestStore) All() ([]Contest, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var contests []Contest
	for _, contest := range s.contests {
		contests = append(contests, contest)
	}
	return contests, nil
}

func (s *ContestStore) reset() {
	s.contests = map[int64]Contest{}
}

func (s *ContestStore) makeObject(id int64) Contest {
	return Contest{ID: id}
}

func (s *ContestStore) makeObjectEvent(typ EventType) ObjectEvent[Contest] {
	return ContestEvent{baseEvent: makeBaseEvent(typ)}
}

func (s *ContestStore) onCreateObject(contest Contest) {
	s.contests[contest.ID] = contest
}

func (s *ContestStore) onDeleteObject(id int64) {
	if contest, ok := s.contests[id]; ok {
		delete(s.contests, contest.ID)
	}
}

// NewContestStore creates a new instance of ContestStore.
func NewContestStore(
	db *gosql.DB, table, eventTable string,
) *ContestStore {
	impl := &ContestStore{}
	impl.baseStore = makeBaseStore[Contest, ContestEvent](
		db, table, eventTable, impl,
	)
	return impl
}

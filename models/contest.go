package models

import (
	"database/sql"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/db"
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
	o.Config = o.Config.clone()
	return o
}

// ContestEvent represents a contest event.
type ContestEvent struct {
	baseEvent
	Contest
}

// Object returns event contest.
func (e ContestEvent) Object() db.Object {
	return e.Contest
}

// WithObject returns event with replaced Contest.
func (e ContestEvent) WithObject(o db.Object) ObjectEvent {
	e.Contest = o.(Contest)
	return e
}

// ContestStore represents store for contests.
type ContestStore struct {
	baseStore
	contests map[int64]Contest
}

// CreateTx creates contest and returns copy with valid ID.
func (s *ContestStore) CreateTx(
	tx gosql.WeakTx, contest Contest,
) (Contest, error) {
	event, err := s.createObjectEvent(tx, ContestEvent{
		makeBaseEvent(CreateEvent),
		contest,
	})
	if err != nil {
		return Contest{}, err
	}
	return event.Object().(Contest), nil
}

// UpdateTx updates contest with specified ID.
func (s *ContestStore) UpdateTx(tx gosql.WeakTx, contest Contest) error {
	_, err := s.createObjectEvent(tx, ContestEvent{
		makeBaseEvent(UpdateEvent),
		contest,
	})
	return err
}

// DeleteTx deletes contest with specified ID.
func (s *ContestStore) DeleteTx(tx gosql.WeakTx, id int64) error {
	_, err := s.createObjectEvent(tx, ContestEvent{
		makeBaseEvent(DeleteEvent),
		Contest{ID: id},
	})
	return err
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

func (s *ContestStore) onCreateObject(o db.Object) {
	contest := o.(Contest)
	s.contests[contest.ID] = contest
}

func (s *ContestStore) onDeleteObject(o db.Object) {
	contest := o.(Contest)
	delete(s.contests, contest.ID)
}

func (s *ContestStore) onUpdateObject(o db.Object) {
	contest := o.(Contest)
	if old, ok := s.contests[contest.ID]; ok {
		s.onDeleteObject(old)
	}
	s.onCreateObject(o)
}

// NewContestStore creates a new instance of ContestStore.
func NewContestStore(
	table, eventTable string, dialect gosql.Dialect,
) *ContestStore {
	impl := &ContestStore{}
	impl.baseStore = makeBaseStore(
		Contest{}, table, ContestEvent{}, eventTable, impl, dialect,
	)
	return impl
}

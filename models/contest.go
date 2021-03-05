package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
)

// Contest represents a contest.
type Contest struct {
	ID     int64 `db:"id"`
	Config JSON  `db:"config"`
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
	tx *sql.Tx, contest Contest,
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
func (s *ContestStore) UpdateTx(tx *sql.Tx, contest Contest) error {
	_, err := s.createObjectEvent(tx, ContestEvent{
		makeBaseEvent(UpdateEvent),
		contest,
	})
	return err
}

// DeleteTx deletes contest with specified ID.
func (s *ContestStore) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := s.createObjectEvent(tx, ContestEvent{
		makeBaseEvent(DeleteEvent),
		Contest{ID: id},
	})
	return err
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
	s.onCreateObject(o)
}

// NewContestStore creates a new instance of ContestStore.
func NewContestStore(
	table, eventTable string, dialect db.Dialect,
) *ContestStore {
	impl := &ContestStore{}
	impl.baseStore = makeBaseStore(
		Contest{}, table, ContestEvent{}, eventTable, impl, dialect,
	)
	return impl
}

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

func (o Contest) clone() Contest {
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

// ContestManager represents manager for contests.
type ContestManager struct {
	baseManager
	contests map[int64]Contest
}

// CreateTx creates contest and returns copy with valid ID.
func (m *ContestManager) CreateTx(
	tx *sql.Tx, contest Contest,
) (Contest, error) {
	event, err := m.createObjectEvent(tx, ContestEvent{
		makeBaseEvent(CreateEvent),
		contest,
	})
	if err != nil {
		return Contest{}, err
	}
	return event.Object().(Contest), nil
}

// UpdateTx updates contest with specified ID.
func (m *ContestManager) UpdateTx(tx *sql.Tx, contest Contest) error {
	_, err := m.createObjectEvent(tx, ContestEvent{
		makeBaseEvent(UpdateEvent),
		contest,
	})
	return err
}

// DeleteTx deletes contest with specified ID.
func (m *ContestManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, ContestEvent{
		makeBaseEvent(DeleteEvent),
		Contest{ID: id},
	})
	return err
}

func (m *ContestManager) reset() {
	m.contests = map[int64]Contest{}
}

func (m *ContestManager) onCreateObject(o db.Object) {
	contest := o.(Contest)
	m.contests[contest.ID] = contest
}

func (m *ContestManager) onDeleteObject(o db.Object) {
	contest := o.(Contest)
	delete(m.contests, contest.ID)
}

func (m *ContestManager) onUpdateObject(o db.Object) {
	m.onCreateObject(o)
}

// NewContestManager creates a new instance of ContestManager.
func NewContestManager(
	table, eventTable string, dialect db.Dialect,
) *ContestManager {
	impl := &ContestManager{}
	impl.baseManager = makeBaseManager(
		Contest{}, table, ContestEvent{}, eventTable, impl, dialect,
	)
	return impl
}

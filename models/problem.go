package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
)

// Problem represents a problem.
type Problem struct {
	ID     int64 `json:"" db:"id"`
	Config JSON  `json:"" db:"config"`
}

// ObjectID return ID of problem.
func (o Problem) ObjectID() int64 {
	return o.ID
}

// ProblemEvent represents a problem event.
type ProblemEvent struct {
	baseEvent
	Problem
}

// Object returns event problem.
func (e ProblemEvent) Object() db.Object {
	return e.Problem
}

// WithObject returns event with replaced Problem.
func (e ProblemEvent) WithObject(o db.Object) ObjectEvent {
	e.Problem = o.(Problem)
	return e
}

type ProblemManager struct {
	baseManager
	problems map[int64]Problem
}

// CreateTx creates problem and returns copy with valid ID.
func (m *ProblemManager) CreateTx(
	tx *sql.Tx, problem Problem,
) (Problem, error) {
	event, err := m.createObjectEvent(tx, ProblemEvent{
		makeBaseEvent(CreateEvent),
		problem,
	})
	if err != nil {
		return Problem{}, err
	}
	return event.Object().(Problem), nil
}

// UpdateTx updates problem with specified ID.
func (m *ProblemManager) UpdateTx(tx *sql.Tx, problem Problem) error {
	_, err := m.createObjectEvent(tx, ProblemEvent{
		makeBaseEvent(UpdateEvent),
		problem,
	})
	return err
}

// DeleteTx deletes problem with specified ID.
func (m *ProblemManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, ProblemEvent{
		makeBaseEvent(DeleteEvent),
		Problem{ID: id},
	})
	return err
}

func (m *ProblemManager) reset() {
	m.problems = map[int64]Problem{}
}

func (m *ProblemManager) onCreateObject(o db.Object) {
	problem := o.(Problem)
	m.problems[problem.ID] = problem
}

func (m *ProblemManager) onDeleteObject(o db.Object) {
	problem := o.(Problem)
	delete(m.problems, problem.ID)
}

func (m *ProblemManager) onUpdateObject(o db.Object) {
	m.onCreateObject(o)
}

// NewProblemManager creates a new instance of ProblemManager.
func NewProblemManager(
	table, eventTable string, dialect db.Dialect,
) *ProblemManager {
	impl := &ProblemManager{}
	impl.baseManager = makeBaseManager(
		Problem{}, table, ProblemEvent{}, eventTable, impl, dialect,
	)
	return impl
}

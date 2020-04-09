package models

import (
	"database/sql"

	"github.com/udovin/solve/db"
)

// ContestProblem represents connection for problems.
type ContestProblem struct {
	// ID contains ID of role.
	ID int64 `db:"id" json:""`
	// ContestID contains ID of contest.
	ContestID int64 `db:"contest_id" json:""`
	// ProblemID contains ID of problem.
	ProblemID int64 `db:"problem_id" json:""`
	// Code contains code of problem.
	Code string `db:"code" json:""`
}

// ObjectID return ID of problem.
func (o ContestProblem) ObjectID() int64 {
	return o.ID
}

func (o ContestProblem) clone() ContestProblem {
	return o
}

// ContestProblemEvent represents problem event.
type ContestProblemEvent struct {
	baseEvent
	ContestProblem
}

// Object returns event role edge.
func (e ContestProblemEvent) Object() db.Object {
	return e.ContestProblem
}

// WithObject returns event with replaced ContestProblem.
func (e ContestProblemEvent) WithObject(o db.Object) ObjectEvent {
	e.ContestProblem = o.(ContestProblem)
	return e
}

// ContestProblemManager represents a problem manager.
type ContestProblemManager struct {
	baseManager
	problems  map[int64]ContestProblem
	byContest indexInt64
}

// Get returns problem by ID.
//
// If there is no problem with specified ID then
// sql.ErrNoRows will be returned.
func (m *ContestProblemManager) Get(id int64) (ContestProblem, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if problem, ok := m.problems[id]; ok {
		return problem.clone(), nil
	}
	return ContestProblem{}, sql.ErrNoRows
}

// FindByContest returns problems by parent ID.
func (m *ContestProblemManager) FindByContest(
	contestID int64,
) ([]ContestProblem, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var problems []ContestProblem
	for id := range m.byContest[contestID] {
		if problem, ok := m.problems[id]; ok {
			problems = append(problems, problem.clone())
		}
	}
	return problems, nil
}

// CreateTx creates problem and returns copy with valid ID.
func (m *ContestProblemManager) CreateTx(
	tx *sql.Tx, problem ContestProblem,
) (ContestProblem, error) {
	event, err := m.createObjectEvent(tx, ContestProblemEvent{
		makeBaseEvent(CreateEvent),
		problem,
	})
	if err != nil {
		return ContestProblem{}, err
	}
	return event.Object().(ContestProblem), nil
}

// UpdateTx updates problem with specified ID.
func (m *ContestProblemManager) UpdateTx(
	tx *sql.Tx, problem ContestProblem,
) error {
	_, err := m.createObjectEvent(tx, ContestProblemEvent{
		makeBaseEvent(UpdateEvent),
		problem,
	})
	return err
}

// DeleteTx deletes problem with specified ID.
func (m *ContestProblemManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, ContestProblemEvent{
		makeBaseEvent(DeleteEvent),
		ContestProblem{ID: id},
	})
	return err
}

func (m *ContestProblemManager) reset() {
	m.problems = map[int64]ContestProblem{}
	m.byContest = indexInt64{}
}

func (m *ContestProblemManager) onCreateObject(o db.Object) {
	problem := o.(ContestProblem)
	m.problems[problem.ID] = problem
	m.byContest.Create(problem.ContestID, problem.ID)
}

func (m *ContestProblemManager) onDeleteObject(o db.Object) {
	problem := o.(ContestProblem)
	m.byContest.Delete(problem.ContestID, problem.ID)
	delete(m.problems, problem.ID)
}

func (m *ContestProblemManager) onUpdateObject(o db.Object) {
	problem := o.(ContestProblem)
	if old, ok := m.problems[problem.ID]; ok {
		if old.ContestID != problem.ContestID {
			m.byContest.Delete(old.ContestID, old.ID)
		}
	}
	m.onCreateObject(o)
}

// NewContestProblemManager creates a new instance of ContestProblemManager.
func NewContestProblemManager(
	table, eventTable string, dialect db.Dialect,
) *ContestProblemManager {
	impl := &ContestProblemManager{}
	impl.baseManager = makeBaseManager(
		ContestProblem{}, table, ContestProblemEvent{}, eventTable, impl, dialect,
	)
	return impl
}

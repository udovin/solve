package models

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/db"
)

type Verdict int

const (
	// Accepted means that solution is correct.
	Accepted Verdict = 1
	// Rejected means that solutios is rejected.
	Rejected Verdict = 2
	// CompilationError means that solution can not compiled.
	CompilationError Verdict = 3
	// TimeLimitExceeded means that solution uses more time than allowed.
	TimeLimitExceeded Verdict = 4
	// MemoryLimitExceeded means that solution uses more memory than allowed.
	MemoryLimitExceeded Verdict = 5
	// RuntimeError means that solution runs incorrectly.
	RuntimeError Verdict = 6
	// WrongAnswer means that solution is incorrect.
	WrongAnswer Verdict = 7
	// PresentationError means that solution output is incorrect.
	PresentationError Verdict = 8
	// PartiallyAccepted means that solution is partially accepted.
	PartiallyAccepted Verdict = 9
)

func (v Verdict) String() string {
	switch v {
	case Accepted:
		return "accepted"
	case Rejected:
		return "rejected"
	case CompilationError:
		return "compilation_error"
	case TimeLimitExceeded:
		return "time_limit_exceeded"
	case MemoryLimitExceeded:
		return "memory_limit_exceeded"
	case RuntimeError:
		return "runtime_error"
	case WrongAnswer:
		return "wrong_answer"
	case PresentationError:
		return "presentation_error"
	case PartiallyAccepted:
		return "partially_accepted"
	default:
		return fmt.Sprintf("Verdict(%d)", v)
	}
}

type UsageReport struct {
	Time   int64 `json:"time"`
	Memory int64 `json:"memory"`
}

type TestReport struct {
	Verdict  Verdict     `json:"verdict"`
	Usage    UsageReport `json:"usage"`
	CheckLog string      `json:"check_log"`
	Points   *float64    `json:"points"`
}

type SolutionReport struct {
	Verdict    Verdict      `json:"verdict"`
	Usage      UsageReport  `json:"usage"`
	CompileLog string       `json:"compile_log"`
	Tests      []TestReport `json:"tests"`
	Points     *float64     `json:"points"`
}

// Solution represents a solution.
type Solution struct {
	ID        int64 `db:"id"`
	ProblemID int64 `db:"problem_id"`
	AuthorID  int64 `db:"author_id"`
	Report    JSON  `db:"report"`
}

// ObjectID return ID of solution.
func (o Solution) ObjectID() int64 {
	return o.ID
}

// Clone creates copy of solution.
func (o Solution) Clone() Solution {
	o.Report = o.Report.Clone()
	return o
}

// GetReport returns solution report.
func (o Solution) GetReport() (*SolutionReport, error) {
	var report *SolutionReport
	err := json.Unmarshal(o.Report, &report)
	return report, err
}

// SetReport sets serialized report to solution.
func (o *Solution) SetReport(report *SolutionReport) error {
	raw, err := json.Marshal(report)
	if err != nil {
		return err
	}
	o.Report = raw
	return nil
}

// SolutionEvent represents a solution event.
type SolutionEvent struct {
	baseEvent
	Solution
}

// Object returns event solution.
func (e SolutionEvent) Object() db.Object {
	return e.Solution
}

// WithObject returns event with replaced Solution.
func (e SolutionEvent) WithObject(o db.Object) ObjectEvent {
	e.Solution = o.(Solution)
	return e
}

// SolutionStore represents store for solutions.
type SolutionStore struct {
	baseStore[Solution, SolutionEvent]
	solutions map[int64]Solution
}

// CreateTx creates solution and returns copy with valid ID.
func (s *SolutionStore) CreateTx(tx gosql.WeakTx, solution *Solution) error {
	event, err := s.createObjectEvent(tx, SolutionEvent{
		makeBaseEvent(CreateEvent), *solution,
	})
	if err != nil {
		return err
	}
	*solution = event.Object().(Solution)
	return nil
}

// UpdateTx updates solution with specified ID.
func (s *SolutionStore) UpdateTx(tx gosql.WeakTx, solution Solution) error {
	_, err := s.createObjectEvent(tx, SolutionEvent{
		makeBaseEvent(UpdateEvent), solution,
	})
	return err
}

// DeleteTx deletes solution with specified ID.
func (s *SolutionStore) DeleteTx(tx gosql.WeakTx, id int64) error {
	_, err := s.createObjectEvent(tx, SolutionEvent{
		makeBaseEvent(DeleteEvent), Solution{ID: id},
	})
	return err
}

// Get returns solution by ID.
//
// If there is no solution with specified ID then
// sql.ErrNoRows will be returned.
func (s *SolutionStore) Get(id int64) (Solution, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if solution, ok := s.solutions[id]; ok {
		return solution.Clone(), nil
	}
	return Solution{}, sql.ErrNoRows
}

// All returns all solutions.
func (s *SolutionStore) All() ([]Solution, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var solutions []Solution
	for _, solution := range s.solutions {
		solutions = append(solutions, solution)
	}
	return solutions, nil
}

func (s *SolutionStore) reset() {
	s.solutions = map[int64]Solution{}
}

func (s *SolutionStore) onCreateObject(solution Solution) {
	s.solutions[solution.ID] = solution
}

func (s *SolutionStore) onDeleteObject(solution Solution) {
	delete(s.solutions, solution.ID)
}

func (s *SolutionStore) onUpdateObject(solution Solution) {
	if old, ok := s.solutions[solution.ID]; ok {
		s.onDeleteObject(old)
	}
	s.onCreateObject(solution)
}

// NewSolutionStore creates a new instance of SolutionStore.
func NewSolutionStore(
	db *gosql.DB, table, eventTable string,
) *SolutionStore {
	impl := &SolutionStore{}
	impl.baseStore = makeBaseStore[Solution, SolutionEvent](
		db, table, eventTable, impl,
	)
	return impl
}

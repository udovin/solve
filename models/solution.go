package models

import (
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
	// Failed means that solution checker is failed.
	Failed Verdict = 10
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
	case Failed:
		return "failed"
	default:
		return fmt.Sprintf("Verdict(%d)", v)
	}
}

func (v Verdict) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *Verdict) UnmarshalText(data []byte) error {
	switch s := string(data); s {
	case "accepted":
		*v = Accepted
	case "rejected":
		*v = Rejected
	case "compilation_error":
		*v = CompilationError
	case "time_limit_exceeded":
		*v = TimeLimitExceeded
	case "memory_limit_exceeded":
		*v = MemoryLimitExceeded
	case "runtime_error":
		*v = RuntimeError
	case "wrong_answer":
		*v = WrongAnswer
	case "presentation_error":
		*v = PresentationError
	case "partially_accepted":
		*v = PartiallyAccepted
	case "failed":
		*v = Failed
	default:
		return fmt.Errorf("unsupported kind: %q", s)
	}
	return nil
}

type UsageReport struct {
	Time   int64 `json:"time,omitempty"`
	Memory int64 `json:"memory,omitempty"`
}

type ExecuteReport struct {
	Usage UsageReport `json:"usage"`
	Log   string      `json:"log"`
}

type TestReport struct {
	Verdict    Verdict        `json:"verdict"`
	Usage      UsageReport    `json:"usage"`
	Checker    *ExecuteReport `json:"checker,omitempty"`
	Interactor *ExecuteReport `json:"interactor,omitempty"`
	Points     *float64       `json:"points,omitempty"`
}

type SolutionReport struct {
	Verdict  Verdict        `json:"verdict"`
	Usage    UsageReport    `json:"usage"`
	Compiler *ExecuteReport `json:"compiler,omitempty"`
	Tests    []TestReport   `json:"tests,omitempty"`
	Points   *float64       `json:"points,omitempty"`
}

// Solution represents a solution.
type Solution struct {
	baseObject
	ProblemID  int64   `db:"problem_id"`
	CompilerID int64   `db:"compiler_id"`
	AuthorID   int64   `db:"author_id"`
	Report     JSON    `db:"report"`
	CreateTime int64   `db:"create_time"`
	Content    NString `db:"content"`
	ContentID  NInt64  `db:"content_id"`
}

// Clone creates copy of solution.
func (o Solution) Clone() Solution {
	o.Report = o.Report.Clone()
	return o
}

// GetReport returns solution report.
func (o Solution) GetReport() (*SolutionReport, error) {
	if o.Report == nil {
		return nil, nil
	}
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
func (e SolutionEvent) Object() Solution {
	return e.Solution
}

// SetObject sets event solution.
func (e *SolutionEvent) SetObject(o Solution) {
	e.Solution = o
}

// SolutionStore represents store for solutions.
type SolutionStore struct {
	cachedStore[Solution, SolutionEvent, *Solution, *SolutionEvent]
	byProblem *index[int64, Solution, *Solution]
}

func (s *SolutionStore) GetEventStore() db.EventStore[SolutionEvent, *SolutionEvent] {
	return s.events
}

func (s *SolutionStore) FindByProblem(id int64) ([]Solution, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []Solution
	for id := range s.byProblem.Get(id) {
		if object, ok := s.objects.Get(id); ok {
			objects = append(objects, object.Clone())
		}
	}
	return objects, nil
}

// NewSolutionStore creates a new instance of SolutionStore.
func NewSolutionStore(
	db *gosql.DB, table, eventTable string,
) *SolutionStore {
	impl := &SolutionStore{
		byProblem: newIndex(func(o Solution) int64 { return o.ProblemID }),
	}
	impl.cachedStore = makeCachedStore[Solution, SolutionEvent](
		db, table, eventTable, impl, impl.byProblem,
	)
	return impl
}

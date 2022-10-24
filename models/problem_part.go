package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

type ProblemPartKind int

const (
	ProblemStatement ProblemPartKind = 1
)

type ProblemStatementConfig struct {
	Locale       string `json:"locale"`
	Title        string `json:"title"`
	Legend       string `json:"legend,omitempty"`
	InputFormat  string `json:"input_format,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

// ProblemPart represents a problem part.
type ProblemPart struct {
	baseObject
	ProblemID int64           `db:"problem_id"`
	Kind      ProblemPartKind `db:"kind"`
	Config    JSON            `db:"config"`
	FileID    NInt64          `db:"file_id"`
}

// Clone creates copy of problem part.
func (o ProblemPart) Clone() ProblemPart {
	return o
}

// ProblemPartEvent represents a problem part event.
type ProblemPartEvent struct {
	baseEvent
	ProblemPart
}

// Object returns event problem part.
func (e ProblemPartEvent) Object() ProblemPart {
	return e.ProblemPart
}

// SetObject sets event problem part.
func (e *ProblemPartEvent) SetObject(o ProblemPart) {
	e.ProblemPart = o
}

// ProblemPartStore represents store for problem parts.
type ProblemPartStore struct {
	baseStore[ProblemPart, ProblemPartEvent, *ProblemPart, *ProblemPartEvent]
	objects   map[int64]ProblemPart
	byProblem index[int64]
}

// Get returns problem part by ID.
//
// If there is no problem part with specified ID then
// sql.ErrNoRows will be returned.
func (s *ProblemPartStore) Get(id int64) (ProblemPart, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if object, ok := s.objects[id]; ok {
		return object.Clone(), nil
	}
	return ProblemPart{}, sql.ErrNoRows
}

func (s *ProblemPartStore) FindByProblem(id int64) ([]ProblemPart, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []ProblemPart
	for id := range s.byProblem[id] {
		if object, ok := s.objects[id]; ok {
			objects = append(objects, object.Clone())
		}
	}
	return objects, nil
}

// All returns all problem parts.
func (s *ProblemPartStore) All() ([]ProblemPart, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []ProblemPart
	for _, object := range s.objects {
		objects = append(objects, object)
	}
	return objects, nil
}

func (s *ProblemPartStore) reset() {
	s.objects = map[int64]ProblemPart{}
	s.byProblem = index[int64]{}
}

func (s *ProblemPartStore) onCreateObject(object ProblemPart) {
	s.objects[object.ID] = object
	s.byProblem.Create(object.ProblemID, object.ID)
}

func (s *ProblemPartStore) onDeleteObject(id int64) {
	if object, ok := s.objects[id]; ok {
		s.byProblem.Delete(object.ProblemID, object.ID)
		delete(s.objects, object.ID)
	}
}

var _ baseStoreImpl[ProblemPart] = (*ProblemPartStore)(nil)

// NewProblemPartStore creates a new instance of ProblemPartStore.
func NewProblemPartStore(
	db *gosql.DB, table, eventTable string,
) *ProblemPartStore {
	impl := &ProblemPartStore{}
	impl.baseStore = makeBaseStore[ProblemPart, ProblemPartEvent](
		db, table, eventTable, impl,
	)
	return impl
}

package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

type ProblemResourceKind int

const (
	ProblemStatement ProblemResourceKind = 1
)

type ProblemStatementConfig struct {
	Locale       string `json:"locale"`
	Title        string `json:"title"`
	Legend       string `json:"legend,omitempty"`
	InputFormat  string `json:"input_format,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

// ProblemResource represents a problem resource.
type ProblemResource struct {
	baseObject
	ProblemID int64               `db:"problem_id"`
	Kind      ProblemResourceKind `db:"kind"`
	Config    JSON                `db:"config"`
	FileID    NInt64              `db:"file_id"`
}

// Clone creates copy of problem resource.
func (o ProblemResource) Clone() ProblemResource {
	return o
}

// ProblemResourceEvent represents a problem resource event.
type ProblemResourceEvent struct {
	baseEvent
	ProblemResource
}

// Object returns event problem resource.
func (e ProblemResourceEvent) Object() ProblemResource {
	return e.ProblemResource
}

// SetObject sets event problem resource.
func (e *ProblemResourceEvent) SetObject(o ProblemResource) {
	e.ProblemResource = o
}

// ProblemResourceStore represents store for problem resources.
type ProblemResourceStore struct {
	baseStore[ProblemResource, ProblemResourceEvent, *ProblemResource, *ProblemResourceEvent]
	objects   map[int64]ProblemResource
	byProblem index[int64]
}

// Get returns problem resource by ID.
//
// If there is no problem resource with specified ID then
// sql.ErrNoRows will be returned.
func (s *ProblemResourceStore) Get(id int64) (ProblemResource, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if object, ok := s.objects[id]; ok {
		return object.Clone(), nil
	}
	return ProblemResource{}, sql.ErrNoRows
}

func (s *ProblemResourceStore) FindByProblem(id int64) ([]ProblemResource, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []ProblemResource
	for id := range s.byProblem[id] {
		if object, ok := s.objects[id]; ok {
			objects = append(objects, object.Clone())
		}
	}
	return objects, nil
}

// All returns all problem resources.
func (s *ProblemResourceStore) All() ([]ProblemResource, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []ProblemResource
	for _, object := range s.objects {
		objects = append(objects, object)
	}
	return objects, nil
}

func (s *ProblemResourceStore) reset() {
	s.objects = map[int64]ProblemResource{}
	s.byProblem = index[int64]{}
}

func (s *ProblemResourceStore) onCreateObject(object ProblemResource) {
	s.objects[object.ID] = object
	s.byProblem.Create(object.ProblemID, object.ID)
}

func (s *ProblemResourceStore) onDeleteObject(id int64) {
	if object, ok := s.objects[id]; ok {
		s.byProblem.Delete(object.ProblemID, object.ID)
		delete(s.objects, object.ID)
	}
}

var _ baseStoreImpl[ProblemResource] = (*ProblemResourceStore)(nil)

// NewProblemResourceStore creates a new instance of ProblemResourceStore.
func NewProblemResourceStore(
	db *gosql.DB, table, eventTable string,
) *ProblemResourceStore {
	impl := &ProblemResourceStore{}
	impl.baseStore = makeBaseStore[ProblemResource, ProblemResourceEvent](
		db, table, eventTable, impl,
	)
	return impl
}
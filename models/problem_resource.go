package models

import (
	"database/sql"
	"encoding/json"

	"github.com/udovin/gosql"
)

type ProblemResourceKind int

const (
	ProblemStatement         ProblemResourceKind = 1
	ProblemStatementResource ProblemResourceKind = 2
)

type ProblemStatementSample struct {
	Input  string `json:"input,omitempty"`
	Output string `json:"output,omitempty"`
}

type ProblemStatementConfig struct {
	Locale string `json:"locale"`
	Title  string `json:"title"`
	Legend string `json:"legend,omitempty"`
	Input  string `json:"input,omitempty"`
	Output string `json:"output,omitempty"`
	Notes  string `json:"notes,omitempty"`
	// Samples contains problem sample tests.
	Samples []ProblemStatementSample `json:"samples,omitempty"`
}

func (c ProblemStatementConfig) ProblemResourceKind() ProblemResourceKind {
	return ProblemStatement
}

type ProblemStatementResourceConfig struct {
	Locale string `json:"locale"`
	Name   string `json:"name"`
}

func (c ProblemStatementResourceConfig) ProblemResourceKind() ProblemResourceKind {
	return ProblemStatementResource
}

type ProblemResourceConfig interface {
	ProblemResourceKind() ProblemResourceKind
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
	o.Config = o.Config.Clone()
	return o
}

func (o ProblemResource) ScanConfig(config ProblemResourceConfig) error {
	return json.Unmarshal(o.Config, config)
}

// SetConfig updates kind and config of task.
func (o *ProblemResource) SetConfig(config ProblemResourceConfig) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	o.Kind = config.ProblemResourceKind()
	o.Config = raw
	return nil
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

//lint:ignore U1000 Used in generic interface.
func (s *ProblemResourceStore) reset() {
	s.objects = map[int64]ProblemResource{}
	s.byProblem = index[int64]{}
}

//lint:ignore U1000 Used in generic interface.
func (s *ProblemResourceStore) onCreateObject(object ProblemResource) {
	s.objects[object.ID] = object
	s.byProblem.Create(object.ProblemID, object.ID)
}

//lint:ignore U1000 Used in generic interface.
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

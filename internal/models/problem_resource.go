package models

import (
	"context"
	"encoding/json"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
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
	Locale      string `json:"locale"`
	Title       string `json:"title"`
	Legend      string `json:"legend,omitempty"`
	Input       string `json:"input,omitempty"`
	Output      string `json:"output,omitempty"`
	Notes       string `json:"notes,omitempty"`
	Scoring     string `json:"scoring,omitempty"`
	Interaction string `json:"interaction,omitempty"`
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
	cachedStore[ProblemResource, ProblemResourceEvent, *ProblemResource, *ProblemResourceEvent]
	byProblem *btreeIndex[int64, ProblemResource, *ProblemResource]
}

func (s *ProblemResourceStore) FindByProblem(
	ctx context.Context, problemID int64,
) (db.Rows[ProblemResource], error) {
	if err := s.TrySync(ctx); err != nil {
		return nil, err
	}
	s.mutex.RLock()
	return btreeIndexFind(
		s.byProblem,
		s.objects.Iter(),
		s.mutex.RLocker(),
		problemID,
	), nil
}

// NewProblemResourceStore creates a new instance of ProblemResourceStore.
func NewProblemResourceStore(
	db *gosql.DB, table, eventTable string,
) *ProblemResourceStore {
	impl := &ProblemResourceStore{
		byProblem: newBTreeIndex(
			func(o ProblemResource) (int64, bool) { return o.ProblemID, true },
			lessInt64,
		),
	}
	impl.cachedStore = makeCachedStore[ProblemResource, ProblemResourceEvent](
		db, table, eventTable, impl, impl.byProblem,
	)
	return impl
}

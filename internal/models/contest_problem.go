package models

import (
	"encoding/json"

	"github.com/udovin/gosql"
)

type ContestProblemConfig struct {
	Points *int `json:"points,omitempty"`
	// Locales contains list of allowed locales.
	Locales []string `json:"locales,omitempty"`
}

// ContestProblem represents connection for problems.
type ContestProblem struct {
	baseObject
	// ProblemID contains ID of problem.
	ProblemID int64 `db:"problem_id"`
	// ContestID contains ID of contest.
	ContestID int64 `db:"contest_id"`
	// Code contains code of problem.
	Code string `db:"code"`
	// Config contains problem config.
	Config JSON `db:"config"`
}

func (o ContestProblem) GetConfig() (ContestProblemConfig, error) {
	var config ContestProblemConfig
	if len(o.Config) == 0 {
		return config, nil
	}
	err := json.Unmarshal(o.Config, &config)
	return config, err
}

func (o *ContestProblem) SetConfig(config ContestProblemConfig) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	o.Config = raw
	return nil
}

// Clone creates copy of contest problem.
func (o ContestProblem) Clone() ContestProblem {
	o.Config = o.Config.Clone()
	return o
}

// ContestProblemEvent represents problem event.
type ContestProblemEvent struct {
	baseEvent
	ContestProblem
}

// Object returns event role edge.
func (e ContestProblemEvent) Object() ContestProblem {
	return e.ContestProblem
}

// SetObject sets event contest problem.
func (e *ContestProblemEvent) SetObject(o ContestProblem) {
	e.ContestProblem = o
}

// ContestProblemStore represents a problem store.
type ContestProblemStore struct {
	cachedStore[ContestProblem, ContestProblemEvent, *ContestProblem, *ContestProblemEvent]
	byContest *index[int64, ContestProblem, *ContestProblem]
}

// FindByContest returns problems by parent ID.
func (s *ContestProblemStore) FindByContest(id int64) ([]ContestProblem, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []ContestProblem
	for id := range s.byContest.Get(id) {
		if object, ok := s.objects.Get(id); ok {
			objects = append(objects, object.Clone())
		}
	}
	return objects, nil
}

// NewContestProblemStore creates a new instance of ContestProblemStore.
func NewContestProblemStore(
	db *gosql.DB, table, eventTable string,
) *ContestProblemStore {
	impl := &ContestProblemStore{
		byContest: newIndex(func(o ContestProblem) int64 { return o.ContestID }),
	}
	impl.cachedStore = makeCachedStore[ContestProblem, ContestProblemEvent](
		db, table, eventTable, impl, impl.byContest,
	)
	return impl
}

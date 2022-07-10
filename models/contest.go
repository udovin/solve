package models

import (
	"database/sql"
	"encoding/json"

	"github.com/udovin/gosql"
)

type ContestConfig struct {
	BeginTime          NInt64 `json:"begin_time"`
	Duration           int    `json:"duration"`
	EnableRegistration bool   `json:"enable_registration"`
	EnableUpsolving    bool   `json:"enable_upsolving"`
}

// Contest represents a contest.
type Contest struct {
	baseObject
	OwnerID NInt64 `db:"owner_id"`
	Config  JSON   `db:"config"`
	Title   string `db:"title"`
}

// Clone creates copy of contest.
func (o Contest) Clone() Contest {
	o.Config = o.Config.Clone()
	return o
}

func (o Contest) GetConfig() (ContestConfig, error) {
	var config ContestConfig
	if len(o.Config) == 0 {
		return config, nil
	}
	err := json.Unmarshal(o.Config, &config)
	return config, err
}

func (o *Contest) SetConfig(config ContestConfig) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	o.Config = raw
	return nil
}

// ContestEvent represents a contest event.
type ContestEvent struct {
	baseEvent
	Contest
}

// Object returns event contest.
func (e ContestEvent) Object() Contest {
	return e.Contest
}

// SetObject sets event contest.
func (e *ContestEvent) SetObject(o Contest) {
	e.Contest = o
}

// ContestStore represents store for contests.
type ContestStore struct {
	baseStore[Contest, ContestEvent, *Contest, *ContestEvent]
	contests map[int64]Contest
}

// Get returns contest by ID.
//
// If there is no contest with specified ID then
// sql.ErrNoRows will be returned.
func (s *ContestStore) Get(id int64) (Contest, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if contest, ok := s.contests[id]; ok {
		return contest.Clone(), nil
	}
	return Contest{}, sql.ErrNoRows
}

// All returns all contests.
func (s *ContestStore) All() ([]Contest, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var contests []Contest
	for _, contest := range s.contests {
		contests = append(contests, contest)
	}
	return contests, nil
}

func (s *ContestStore) reset() {
	s.contests = map[int64]Contest{}
}

func (s *ContestStore) onCreateObject(contest Contest) {
	s.contests[contest.ID] = contest
}

func (s *ContestStore) onDeleteObject(id int64) {
	if contest, ok := s.contests[id]; ok {
		delete(s.contests, contest.ID)
	}
}

var _ baseStoreImpl[Contest] = (*ContestStore)(nil)

// NewContestStore creates a new instance of ContestStore.
func NewContestStore(
	db *gosql.DB, table, eventTable string,
) *ContestStore {
	impl := &ContestStore{}
	impl.baseStore = makeBaseStore[Contest, ContestEvent](
		db, table, eventTable, impl,
	)
	return impl
}

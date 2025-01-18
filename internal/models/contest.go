package models

import (
	"encoding/json"
	"fmt"

	"github.com/udovin/gosql"
)

type StandingsKind int

const (
	DisabledStandings StandingsKind = 0
	ICPCStandings     StandingsKind = 1
	IOIStandings      StandingsKind = 2
)

func (v StandingsKind) String() string {
	switch v {
	case DisabledStandings:
		return "disabled"
	case ICPCStandings:
		return "icpc"
	case IOIStandings:
		return "ioi"
	default:
		return fmt.Sprintf("StandingsKind(%d)", v)
	}
}

func (v StandingsKind) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *StandingsKind) UnmarshalText(data []byte) error {
	switch s := string(data); s {
	case "disabled":
		*v = DisabledStandings
	case "icpc":
		*v = ICPCStandings
	case "ioi":
		*v = IOIStandings
	default:
		return fmt.Errorf("unsupported kind: %q", s)
	}
	return nil
}

type ContestConfig struct {
	BeginTime           NInt64        `json:"begin_time,omitempty"`
	Duration            int           `json:"duration,omitempty"`
	EnableRegistration  bool          `json:"enable_registration"`
	EnableVirtual       bool          `json:"enable_virtual"`
	EnableUpsolving     bool          `json:"enable_upsolving"`
	EnableObserving     bool          `json:"enable_observing,omitempty"`
	FreezeBeginDuration int           `json:"freeze_begin_duration,omitempty"`
	FreezeEndTime       NInt64        `json:"freeze_end_time,omitempty"`
	StandingsKind       StandingsKind `json:"standings_kind,omitempty"`
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
	cachedStore[Contest, ContestEvent, *Contest, *ContestEvent]
}

// NewContestStore creates a new instance of ContestStore.
func NewContestStore(
	db *gosql.DB, table, eventTable string,
) *ContestStore {
	impl := &ContestStore{}
	impl.cachedStore = makeCachedStore[Contest, ContestEvent](
		db, table, eventTable, impl,
	)
	return impl
}

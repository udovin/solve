package models

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
)

type ParticipantKind int

const (
	RegularParticipant   ParticipantKind = 1
	UpsolvingParticipant ParticipantKind = 2
	ManagerParticipant   ParticipantKind = 3
	ObserverParticipant  ParticipantKind = 4
)

// String returns string representation.
func (k ParticipantKind) String() string {
	switch k {
	case RegularParticipant:
		return "regular"
	case UpsolvingParticipant:
		return "upsolving"
	case ManagerParticipant:
		return "manager"
	case ObserverParticipant:
		return "observer"
	default:
		return fmt.Sprintf("ParticipantKind(%d)", k)
	}
}

func (k ParticipantKind) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}

func (k *ParticipantKind) UnmarshalText(data []byte) error {
	switch s := string(data); s {
	case "regular":
		*k = RegularParticipant
	case "upsolving":
		*k = UpsolvingParticipant
	case "manager":
		*k = ManagerParticipant
	case "observer":
		*k = ObserverParticipant
	default:
		return fmt.Errorf("unsupported kind: %q", s)
	}
	return nil
}

type RegularParticipantConfig struct {
	BeginTime NInt64 `json:"begin_time,omitempty"`
}

// ContestParticipant represents participant.
type ContestParticipant struct {
	baseObject
	// ContestID contains ID of contest.
	ContestID int64 `db:"contest_id"`
	// AccountID contains ID of account.
	AccountID int64 `db:"account_id"`
	// Kind contains participant kind.
	Kind ParticipantKind `db:"kind"`
	// Config contains participant config.
	Config JSON `db:"config"`
}

// Clone creates copy of contest participant.
func (o ContestParticipant) Clone() ContestParticipant {
	o.Config = o.Config.Clone()
	return o
}

func (o ContestParticipant) ScanConfig(config any) error {
	if len(o.Config) == 0 {
		return nil
	}
	return json.Unmarshal(o.Config, config)
}

func (o *ContestParticipant) SetConfig(config any) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	o.Config = raw
	return nil
}

// ContestParticipant represents participant event.
type ContestParticipantEvent struct {
	baseEvent
	ContestParticipant
}

// Object returns event participant.
func (e ContestParticipantEvent) Object() ContestParticipant {
	return e.ContestParticipant
}

// SetObject sets event contest participant.
func (e *ContestParticipantEvent) SetObject(o ContestParticipant) {
	e.ContestParticipant = o
}

// ContestParticipantStore represents a participant store.
type ContestParticipantStore struct {
	cachedStore[ContestParticipant, ContestParticipantEvent, *ContestParticipant, *ContestParticipantEvent]
	byContest        *btreeIndex[int64, ContestParticipant, *ContestParticipant]
	byContestAccount *btreeIndex[pair[int64, int64], ContestParticipant, *ContestParticipant]
}

func (s *ContestParticipantStore) FindByContest(
	ctx context.Context, contestID int64,
) (db.Rows[ContestParticipant], error) {
	s.mutex.RLock()
	return btreeIndexFind(
		s.byContest,
		s.objects.Iter(),
		s.mutex.RLocker(),
		contestID,
	), nil
}

// FindByContestAccount returns participants by contest and account.
func (s *ContestParticipantStore) FindByContestAccount(
	ctx context.Context, contestID int64, accountID int64,
) (db.Rows[ContestParticipant], error) {
	s.mutex.RLock()
	return btreeIndexFind(
		s.byContestAccount,
		s.objects.Iter(),
		s.mutex.RLocker(),
		makePair(contestID, accountID),
	), nil
}

// NewContestParticipantStore creates a new instance of
// ContestParticipantStore.
func NewContestParticipantStore(
	db *gosql.DB, table, eventTable string,
) *ContestParticipantStore {
	impl := &ContestParticipantStore{
		byContest: newBTreeIndex(func(o ContestParticipant) (int64, bool) { return o.ContestID, true }, lessInt64),
		byContestAccount: newBTreeIndex(func(o ContestParticipant) (pair[int64, int64], bool) {
			return makePair(o.ContestID, o.AccountID), true
		}, lessPairInt64),
	}
	impl.cachedStore = makeCachedStore[ContestParticipant, ContestParticipantEvent](
		db, table, eventTable, impl, impl.byContest, impl.byContestAccount,
	)
	return impl
}

package models

import (
	"fmt"

	"github.com/udovin/gosql"
)

type ParticipantKind int

const (
	RegularParticipant   ParticipantKind = 1
	UpsolvingParticipant ParticipantKind = 2
	ManagerParticipant   ParticipantKind = 3
	VirtualParticipant   ParticipantKind = 4
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
	default:
		return fmt.Errorf("unsupported kind: %q", s)
	}
	return nil
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
	baseStore[ContestParticipant, ContestParticipantEvent, *ContestParticipant, *ContestParticipantEvent]
	byContest        *index[int64, ContestParticipant, *ContestParticipant]
	byContestAccount *index[pair[int64, int64], ContestParticipant, *ContestParticipant]
}

func (s *ContestParticipantStore) FindByContest(
	id int64,
) ([]ContestParticipant, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []ContestParticipant
	for id := range s.byContest.Get(id) {
		if object, ok := s.objects[id]; ok {
			objects = append(objects, object.Clone())
		}
	}
	return objects, nil
}

// FindByContestAccount returns participants by contest and account.
func (s *ContestParticipantStore) FindByContestAccount(
	contestID int64, accountID int64,
) ([]ContestParticipant, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var objects []ContestParticipant
	for id := range s.byContestAccount.Get(makePair(contestID, accountID)) {
		if object, ok := s.objects[id]; ok {
			objects = append(objects, object.Clone())
		}
	}
	return objects, nil
}

var _ baseStoreImpl[ContestParticipant] = (*ContestParticipantStore)(nil)

// NewContestParticipantStore creates a new instance of
// ContestParticipantStore.
func NewContestParticipantStore(
	db *gosql.DB, table, eventTable string,
) *ContestParticipantStore {
	impl := &ContestParticipantStore{
		byContest: newIndex(func(o ContestParticipant) int64 { return o.ContestID }),
		byContestAccount: newIndex(func(o ContestParticipant) pair[int64, int64] {
			return makePair(o.ContestID, o.AccountID)
		}),
	}
	impl.baseStore = makeBaseStore[ContestParticipant, ContestParticipantEvent](
		db, table, eventTable, impl, impl.byContest, impl.byContestAccount,
	)
	return impl
}

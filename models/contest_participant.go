package models

import (
	"database/sql"
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
	// ID contains ID of participant.
	ID int64 `db:"id"`
	// ContestID contains ID of contest.
	ContestID int64 `db:"contest_id"`
	// AccountID contains ID of account.
	AccountID int64 `db:"account_id"`
	// Kind contains participant kind.
	Kind ParticipantKind `db:"kind"`
	// Config contains participant config.
	Config JSON `db:"config"`
}

// ObjectID return ID of contest participant.
func (o ContestParticipant) ObjectID() int64 {
	return o.ID
}

// SetObjectID sets ID of contest participant.
func (o *ContestParticipant) SetObjectID(id int64) {
	o.ID = id
}

// Clone creates copy of contest participant.
func (o ContestParticipant) Clone() ContestParticipant {
	o.Config = o.Config.Clone()
	return o
}

func (o ContestParticipant) contestAccountKey() pair[int64, int64] {
	return makePair(o.ContestID, o.AccountID)
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
	participants     map[int64]ContestParticipant
	byContest        index[int64]
	byContestAccount index[pair[int64, int64]]
}

// Get returns participant by ID.
//
// If there is no participant with specified ID then
// sql.ErrNoRows will be returned.
func (s *ContestParticipantStore) Get(id int64) (ContestParticipant, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if participant, ok := s.participants[id]; ok {
		return participant.Clone(), nil
	}
	return ContestParticipant{}, sql.ErrNoRows
}

func (s *ContestParticipantStore) FindByContest(
	contestID int64,
) ([]ContestParticipant, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var participants []ContestParticipant
	for id := range s.byContest[contestID] {
		if participant, ok := s.participants[id]; ok {
			participants = append(participants, participant.Clone())
		}
	}
	return participants, nil
}

// FindByContestAccount returns participants by contest and account.
func (s *ContestParticipantStore) FindByContestAccount(
	contestID int64, accountID int64,
) ([]ContestParticipant, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var participants []ContestParticipant
	for id := range s.byContestAccount[makePair(contestID, accountID)] {
		if participant, ok := s.participants[id]; ok {
			participants = append(participants, participant.Clone())
		}
	}
	return participants, nil
}

func (s *ContestParticipantStore) reset() {
	s.participants = map[int64]ContestParticipant{}
	s.byContest = makeIndex[int64]()
	s.byContestAccount = makeIndex[pair[int64, int64]]()
}

func (s *ContestParticipantStore) makeObject(id int64) ContestParticipant {
	return ContestParticipant{ID: id}
}

func (s *ContestParticipantStore) makeObjectEvent(typ EventType) ContestParticipantEvent {
	return ContestParticipantEvent{baseEvent: makeBaseEvent(typ)}
}

func (s *ContestParticipantStore) onCreateObject(participant ContestParticipant) {
	s.participants[participant.ID] = participant
	s.byContest.Create(participant.ContestID, participant.ID)
	s.byContestAccount.Create(participant.contestAccountKey(), participant.ID)
}

func (s *ContestParticipantStore) onDeleteObject(id int64) {
	if participant, ok := s.participants[id]; ok {
		s.byContest.Delete(participant.ContestID, participant.ID)
		s.byContestAccount.Delete(participant.contestAccountKey(), participant.ID)
		delete(s.participants, participant.ID)
	}
}

// NewContestParticipantStore creates a new instance of
// ContestParticipantStore.
func NewContestParticipantStore(
	db *gosql.DB, table, eventTable string,
) *ContestParticipantStore {
	impl := &ContestParticipantStore{}
	impl.baseStore = makeBaseStore[ContestParticipant, ContestParticipantEvent, *ContestParticipant, *ContestParticipantEvent](
		db, table, eventTable, impl,
	)
	return impl
}

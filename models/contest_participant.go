package models

import (
	"database/sql"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/db"
)

// ContestParticipant represents participant.
type ContestParticipant struct {
	// ID contains ID of participant.
	ID int64 `db:"id"`
	// ContestID contains ID of contest.
	ContestID int64 `db:"contest_id"`
	// AccountID contains ID of account.
	AccountID int64 `db:"account_id"`
}

// ObjectID return ID of problem.
func (o ContestParticipant) ObjectID() int64 {
	return o.ID
}

// Clone creates copy of participant.
func (o ContestParticipant) Clone() ContestParticipant {
	return o
}

func (o ContestParticipant) contestAccountKey() pairInt64 {
	return pairInt64{o.ContestID, o.AccountID}
}

// ContestParticipant represents participant event.
type ContestParticipantEvent struct {
	baseEvent
	ContestParticipant
}

// Object returns event participant.
func (e ContestParticipantEvent) Object() db.Object {
	return e.ContestParticipant
}

// WithObject returns event with replaced ContestParticipant.
func (e ContestParticipantEvent) WithObject(o db.Object) ObjectEvent {
	e.ContestParticipant = o.(ContestParticipant)
	return e
}

// ContestParticipantStore represents a participant store.
type ContestParticipantStore struct {
	baseStore
	participants     map[int64]ContestParticipant
	byContestAccount indexPairInt64
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

// FindByContestAccount returns participants by contest and account.
func (s *ContestParticipantStore) FindByContestAccount(
	contestID int64, accountID int64,
) ([]ContestParticipant, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var participants []ContestParticipant
	for id := range s.byContestAccount[pairInt64{contestID, accountID}] {
		if participant, ok := s.participants[id]; ok {
			participants = append(participants, participant.Clone())
		}
	}
	return participants, nil
}

// CreateTx creates participant and returns copy with valid ID.
func (s *ContestParticipantStore) CreateTx(
	tx gosql.WeakTx, participant *ContestParticipant,
) error {
	event, err := s.createObjectEvent(tx, ContestParticipantEvent{
		makeBaseEvent(CreateEvent),
		*participant,
	})
	if err != nil {
		return err
	}
	*participant = event.Object().(ContestParticipant)
	return nil
}

// UpdateTx updates participant with specified ID.
func (s *ContestParticipantStore) UpdateTx(
	tx gosql.WeakTx, participant ContestParticipant,
) error {
	_, err := s.createObjectEvent(tx, ContestParticipantEvent{
		makeBaseEvent(UpdateEvent),
		participant,
	})
	return err
}

// DeleteTx deletes participant with specified ID.
func (s *ContestParticipantStore) DeleteTx(tx gosql.WeakTx, id int64) error {
	_, err := s.createObjectEvent(tx, ContestParticipantEvent{
		makeBaseEvent(DeleteEvent),
		ContestParticipant{ID: id},
	})
	return err
}

func (s *ContestParticipantStore) reset() {
	s.participants = map[int64]ContestParticipant{}
	s.byContestAccount = indexPairInt64{}
}

func (s *ContestParticipantStore) onCreateObject(o db.Object) {
	participant := o.(ContestParticipant)
	s.participants[participant.ID] = participant
	s.byContestAccount.Create(participant.contestAccountKey(), participant.ID)
}

func (s *ContestParticipantStore) onDeleteObject(o db.Object) {
	participant := o.(ContestParticipant)
	s.byContestAccount.Delete(participant.contestAccountKey(), participant.ID)
	delete(s.participants, participant.ID)
}

func (s *ContestParticipantStore) onUpdateObject(o db.Object) {
	participant := o.(ContestParticipant)
	if old, ok := s.participants[participant.ID]; ok {
		s.onDeleteObject(old)
	}
	s.onCreateObject(o)
}

// NewContestParticipantStore creates a new instance of
// ContestParticipantStore.
func NewContestParticipantStore(
	table, eventTable string, dialect gosql.Dialect,
) *ContestParticipantStore {
	impl := &ContestParticipantStore{}
	impl.baseStore = makeBaseStore(
		ContestParticipant{}, table, ContestParticipantEvent{}, eventTable,
		impl, dialect,
	)
	return impl
}

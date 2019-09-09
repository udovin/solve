package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Participant struct {
	ID         int64 `json:"" db:"id"`
	Type       int8  `json:"" db:"type"`
	ContestID  int64 `json:"" db:"contest_id"`
	UserID     int64 `json:"" db:"user_id"`
	CreateTime int64 `json:"" db:"create_time"`
}

type participantChange struct {
	BaseChange
	Participant
}

type contestUserPair struct {
	ContestID int64
	UserID    int64
}

type ParticipantStore struct {
	Manager      *ChangeManager
	table        string
	changeTable  string
	participants map[int64]Participant
	// contestUser contains map for (contest.ID, user.ID) tuple
	contestUser map[contestUserPair]map[int64]struct{}
	// mutex contains rw mutex
	mutex sync.RWMutex
}

func NewParticipantStore(db *sql.DB, table, changeTable string) *ParticipantStore {
	store := ParticipantStore{
		table:        table,
		changeTable:  changeTable,
		participants: make(map[int64]Participant),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *ParticipantStore) Get(id int64) (Participant, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	participant, ok := s.participants[id]
	return participant, ok
}

func (s *ParticipantStore) GetByContestUser(
	contestID, userID int64,
) []Participant {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	key := contestUserPair{
		ContestID: contestID,
		UserID:    userID,
	}
	if ids, ok := s.contestUser[key]; ok {
		var participants []Participant
		for id := range ids {
			if participant, ok := s.participants[id]; ok {
				participants = append(participants, participant)
			}
		}
		return participants
	}
	return nil
}

func (s *ParticipantStore) Create(m *Participant) error {
	change := participantChange{
		BaseChange:  BaseChange{Type: CreateChange},
		Participant: *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Participant
	return nil
}

func (s *ParticipantStore) CreateTx(tx *ChangeTx, m *Participant) error {
	change := participantChange{
		BaseChange:  BaseChange{Type: CreateChange},
		Participant: *m,
	}
	err := s.Manager.ChangeTx(tx, &change)
	if err != nil {
		return err
	}
	*m = change.Participant
	return nil
}

func (s *ParticipantStore) Update(m *Participant) error {
	change := participantChange{
		BaseChange:  BaseChange{Type: UpdateChange},
		Participant: *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Participant
	return nil
}

func (s *ParticipantStore) Delete(id int64) error {
	change := participantChange{
		BaseChange:  BaseChange{Type: DeleteChange},
		Participant: Participant{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *ParticipantStore) GetLocker() sync.Locker {
	return &s.mutex
}

func (s *ParticipantStore) InitChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *ParticipantStore) LoadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "type", "contest_id", "user_id", "create_time"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *ParticipantStore) ScanChange(scan Scanner) (Change, error) {
	participant := participantChange{}
	err := scan.Scan(
		&participant.BaseChange.ID, &participant.BaseChange.Type,
		&participant.Time, &participant.Participant.ID,
		&participant.Participant.Type, &participant.ContestID,
		&participant.UserID, &participant.CreateTime,
	)
	return &participant, err
}

func (s *ParticipantStore) SaveChange(tx *sql.Tx, change Change) error {
	participant := change.(*participantChange)
	participant.Time = time.Now().Unix()
	switch participant.BaseChange.Type {
	case CreateChange:
		participant.Participant.CreateTime = participant.Time
		var err error
		participant.Participant.ID, err = execTxReturningID(
			s.Manager.db.Driver(), tx,
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("type", "contest_id", "user_id", "create_time")`+
					` VALUES ($1, $2, $3, $4)`,
				s.table,
			),
			"id",
			participant.Participant.Type, participant.ContestID,
			participant.UserID, participant.CreateTime,
		)
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.participants[participant.Participant.ID]; !ok {
			return fmt.Errorf(
				"participant with id = %d does not exists",
				participant.Participant.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET`+
					` "type" = $1, "contest_id" = $2, "user_id" = $3,`+
					` "create_time" = $4`+
					` WHERE "id" = $5`,
				s.table,
			),
			participant.Participant.Type, participant.ContestID,
			participant.UserID, participant.CreateTime,
			participant.Participant.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.participants[participant.Participant.ID]; !ok {
			return fmt.Errorf(
				"participant with id = %d does not exists",
				participant.Participant.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1`,
				s.table,
			),
			participant.Participant.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			participant.BaseChange.Type,
		)
	}
	var err error
	participant.BaseChange.ID, err = execTxReturningID(
		s.Manager.db.Driver(), tx,
		fmt.Sprintf(
			`INSERT INTO "%s"`+
				` ("change_type", "change_time",`+
				` "id", "type", "contest_id", "user_id", "create_time")`+
				` VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			s.changeTable,
		),
		"change_id",
		participant.BaseChange.Type, participant.Time,
		participant.Participant.ID, participant.Participant.Type,
		participant.ContestID, participant.UserID, participant.CreateTime,
	)
	return err
}

func (s *ParticipantStore) ApplyChange(change Change) {
	participant := change.(*participantChange)
	contestUser := contestUserPair{
		ContestID: participant.ContestID,
		UserID:    participant.UserID,
	}
	switch participant.BaseChange.Type {
	case UpdateChange:
		if old, ok := s.participants[participant.Participant.ID]; ok {
			oldKey := contestUserPair{
				ContestID: old.ContestID,
				UserID:    old.UserID,
			}
			if oldKey != contestUser {
				if fields, ok := s.contestUser[contestUser]; ok {
					delete(fields, old.ID)
					if len(fields) == 0 {
						delete(s.contestUser, oldKey)
					}
				}
			}
		}
		fallthrough
	case CreateChange:
		if _, ok := s.contestUser[contestUser]; !ok {
			s.contestUser[contestUser] = make(map[int64]struct{})
		}
		s.contestUser[contestUser][participant.Participant.ID] = struct{}{}
		s.participants[participant.Participant.ID] = participant.Participant
	case DeleteChange:
		if fields, ok := s.contestUser[contestUser]; ok {
			delete(fields, participant.Participant.ID)
			if len(fields) == 0 {
				delete(s.contestUser, contestUser)
			}
		}
		delete(s.participants, participant.Participant.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			participant.BaseChange.Type,
		))
	}
}

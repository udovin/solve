package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Contest struct {
	ID         int64  `json:"" db:"id"`
	UserID     int64  `json:"" db:"user_id"`
	CreateTime int64  `json:"" db:"create_time"`
	Title      string `json:"" db:"title"`
}

type contestChange struct {
	BaseChange
	Contest
}

type ContestStore struct {
	Manager     *ChangeManager
	table       string
	changeTable string
	contests    map[int64]Contest
	mutex       sync.RWMutex
}

func NewContestStore(db *sql.DB, table, changeTable string) *ContestStore {
	store := ContestStore{
		table:       table,
		changeTable: changeTable,
		contests:    make(map[int64]Contest),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *ContestStore) All() ([]Contest, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var result []Contest
	for _, contest := range s.contests {
		result = append(result, contest)
	}
	return result, nil
}

func (s *ContestStore) Get(id int64) (Contest, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if contest, ok := s.contests[id]; ok {
		return contest, nil
	}
	return Contest{}, sql.ErrNoRows
}

func (s *ContestStore) Create(m *Contest) error {
	change := contestChange{
		BaseChange: BaseChange{Type: CreateChange},
		Contest:    *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Contest
	return nil
}

func (s *ContestStore) Update(m *Contest) error {
	change := contestChange{
		BaseChange: BaseChange{Type: UpdateChange},
		Contest:    *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Contest
	return nil
}

func (s *ContestStore) Delete(id int64) error {
	change := contestChange{
		BaseChange: BaseChange{Type: DeleteChange},
		Contest:    Contest{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *ContestStore) GetLocker() sync.Locker {
	return &s.mutex
}

func (s *ContestStore) InitChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *ContestStore) LoadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "user_id", "create_time", "title"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *ContestStore) ScanChange(scan Scanner) (Change, error) {
	contest := contestChange{}
	err := scan.Scan(
		&contest.BaseChange.ID, &contest.Type, &contest.Time,
		&contest.Contest.ID, &contest.UserID, &contest.CreateTime,
		&contest.Title,
	)
	return &contest, err
}

func (s *ContestStore) SaveChange(tx *sql.Tx, change Change) error {
	contest := change.(*contestChange)
	contest.Time = time.Now().Unix()
	switch contest.Type {
	case CreateChange:
		contest.Contest.CreateTime = contest.Time
		var err error
		contest.Contest.ID, err = execTxReturningID(
			s.Manager.db.Driver(), tx,
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("user_id", "create_time", "title")`+
					` VALUES ($1, $2, $3)`,
				s.table,
			),
			"id",
			contest.UserID, contest.CreateTime, contest.Title,
		)
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.contests[contest.Contest.ID]; !ok {
			return fmt.Errorf(
				"contest with id = %d does not exists",
				contest.Contest.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET "user_id" = $1, "title" = $2`+
					` WHERE "id" = $3`,
				s.table,
			),
			contest.UserID, contest.Title, contest.Contest.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.contests[contest.Contest.ID]; !ok {
			return fmt.Errorf(
				"contest with id = %d does not exists",
				contest.Contest.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1`,
				s.table,
			),
			contest.Contest.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			contest.Type,
		)
	}
	var err error
	contest.BaseChange.ID, err = execTxReturningID(
		s.Manager.db.Driver(), tx,
		fmt.Sprintf(
			`INSERT INTO "%s"`+
				` ("change_type", "change_time",`+
				` "id", "user_id", "create_time", "title")`+
				` VALUES ($1, $2, $3, $4, $5, $6)`,
			s.changeTable,
		),
		"change_id",
		contest.Type, contest.Time,
		contest.Contest.ID, contest.UserID, contest.CreateTime,
		contest.Title,
	)
	return err
}

func (s *ContestStore) ApplyChange(change Change) {
	contest := change.(*contestChange)
	switch contest.Type {
	case UpdateChange:
		fallthrough
	case CreateChange:
		s.contests[contest.Contest.ID] = contest.Contest
	case DeleteChange:
		delete(s.contests, contest.Contest.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			contest.Type,
		))
	}
}

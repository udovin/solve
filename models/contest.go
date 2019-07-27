package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Contest struct {
	ID         int64 `json:"" db:"id"`
	OwnerID    int64 `json:"" db:"owner_id"`
	CreateTime int64 `json:"" db:"create_time"`
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

func (s *ContestStore) Get(id int64) (Contest, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	contest, ok := s.contests[id]
	return contest, ok
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

func (s *ContestStore) getLocker() sync.Locker {
	return &s.mutex
}

func (s *ContestStore) initChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *ContestStore) loadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "owner_id", "create_time"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *ContestStore) scanChange(scan Scanner) (Change, error) {
	contest := contestChange{}
	err := scan.Scan(
		&contest.BaseChange.ID, &contest.Type, &contest.Time,
		&contest.Contest.ID, &contest.OwnerID, &contest.CreateTime,
	)
	return &contest, err
}

func (s *ContestStore) saveChange(tx *sql.Tx, change Change) error {
	contest := change.(*contestChange)
	contest.Time = time.Now().Unix()
	switch contest.Type {
	case CreateChange:
		contest.Contest.CreateTime = contest.Time
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("owner_id", "create_time")`+
					` VALUES ($1, $2)`,
				s.table,
			),
			contest.OwnerID, contest.CreateTime,
		)
		if err != nil {
			return err
		}
		contest.Contest.ID, err = res.LastInsertId()
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
				`UPDATE "%s" SET "owner_id" = $1 WHERE "id" = $2`,
				s.table,
			),
			contest.OwnerID, contest.Contest.ID,
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
	res, err := tx.Exec(
		fmt.Sprintf(
			`INSERT INTO "%s"`+
				` ("change_type", "change_time",`+
				` "id", "owner_id", "create_time")`+
				` VALUES ($1, $2, $3, $4, $5)`,
			s.changeTable,
		),
		contest.Type, contest.Time,
		contest.Contest.ID, contest.OwnerID, contest.CreateTime,
	)
	if err != nil {
		return err
	}
	contest.BaseChange.ID, err = res.LastInsertId()
	return err
}

func (s *ContestStore) applyChange(change Change) {
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

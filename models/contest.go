package models

import (
	"database/sql"
	"fmt"
	"time"
)

type Contest struct {
	ID         int64 `db:"id"          json:""`
	OwnerID    int64 `db:"owner_id"    json:""`
	CreateTime int64 `db:"create_time" json:""`
}

type ContestChange struct {
	BaseChange
	Contest
}

type ContestStore struct {
	Manager     *ChangeManager
	db          *sql.DB
	table       string
	changeTable string
	contests    map[int64]Contest
}

func NewContestStore(
	db *sql.DB, table, changeTable string,
) *ContestStore {
	store := ContestStore{
		db: db, table: table, changeTable: changeTable,
		contests: make(map[int64]Contest),
	}
	store.Manager = NewChangeManager(&store)
	return &store
}

func (s *ContestStore) GetDB() *sql.DB {
	return s.db
}

func (s *ContestStore) ChangeTableName() string {
	return s.changeTable
}

func (s *ContestStore) Get(id int64) (Contest, bool) {
	contest, ok := s.contests[id]
	return contest, ok
}

func (s *ContestStore) Create(m *Contest) error {
	change := ContestChange{
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
	change := ContestChange{
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
	change := ContestChange{
		BaseChange: BaseChange{Type: DeleteChange},
		Contest:    Contest{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *ContestStore) loadChangeGapTx(
	tx *ChangeTx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "owner_id", "create_time"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.ChangeTableName(),
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *ContestStore) scanChange(scan Scanner) (Change, error) {
	change := &ContestChange{}
	err := scan.Scan(
		&change.BaseChange.ID, &change.Type, &change.Time,
		&change.Contest.ID, &change.OwnerID, &change.CreateTime,
	)
	if err != nil {
		return nil, err
	}
	return change, nil
}

func (s *ContestStore) saveChangeTx(tx *ChangeTx, change Change) error {
	contest := change.(*ContestChange)
	contest.Time = time.Now().Unix()
	switch contest.Type {
	case CreateChange:
		contest.Contest.CreateTime = contest.Time
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s" `+
					`("owner_id", "create_time") `+
					`VALUES ($1, $2)`,
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
			`INSERT INTO "%s" `+
				`("change_type", "change_time", `+
				`"id", "owner_id", "create_time") `+
				`VALUES ($1, $2, $3, $4, $5)`,
			s.ChangeTableName(),
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
	contestChange := change.(*ContestChange)
	contest := contestChange.Contest
	switch contestChange.Type {
	case UpdateChange:
		fallthrough
	case CreateChange:
		s.contests[contest.ID] = contest
	case DeleteChange:
		delete(s.contests, contest.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			contestChange.Type,
		))
	}
}

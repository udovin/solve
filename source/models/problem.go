package models

import (
	"database/sql"
	"fmt"
)

type Problem struct {
	ID         int64 `db:"id"          json:""`
	OwnerID    int64 `db:"owner_id"    json:""`
	CreateTime int64 `db:"create_time" json:""`
}

type ProblemChange struct {
	Problem
	ChangeBase
}

type ProblemStore struct {
	Manager     *ChangeManager
	db          *sql.DB
	table       string
	changeTable string
	problems    map[int64]Problem
}

func (c *ProblemChange) ChangeData() interface{} {
	return c.Problem
}

func NewProblemStore(
	db *sql.DB, table, changeTable string,
) *ProblemStore {
	store := ProblemStore{
		db: db, table: table, changeTable: changeTable,
		problems: make(map[int64]Problem),
	}
	store.Manager = NewChangeManager(&store)
	return &store
}

func (s *ProblemStore) GetDB() *sql.DB {
	return s.db
}

func (s *ProblemStore) ChangeTableName() string {
	return s.changeTable
}

func (s *ProblemStore) Create(m *Problem) error {
	change, err := s.Manager.Change(CreateChange, *m)
	if err != nil {
		return err
	}
	*m = change.ChangeData().(Problem)
	return nil
}

func (s *ProblemStore) Update(m *Problem) error {
	change, err := s.Manager.Change(UpdateChange, *m)
	if err != nil {
		return err
	}
	*m = change.ChangeData().(Problem)
	return nil
}

func (s *ProblemStore) Delete(id int64) error {
	_, err := s.Manager.Change(UpdateChange, id)
	return err
}

func (s *ProblemStore) scanChange(scan RowScan) (Change, error) {
	change := &ProblemChange{}
	err := scan.Scan(
		&change.ChangeBase.ID, &change.Type, &change.Time,
		&change.Problem.ID, &change.OwnerID,
		&change.CreateTime,
	)
	if err != nil {
		return nil, err
	}
	return change, nil
}

func (s *ProblemStore) createChangeTx(
	tx *sql.Tx, changeType ChangeType, changeTime int64, data interface{},
) (Change, error) {
	var problem Problem
	switch changeType {
	case CreateChange:
		problem = data.(Problem)
		problem.CreateTime = changeTime
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s" `+
					`("owner_id", "create_time") `+
					`VALUES ($1, $2)`,
				s.table,
			),
			problem.OwnerID, problem.CreateTime,
		)
		if err != nil {
			return nil, err
		}
		problemID, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		problem.ID = problemID
	case UpdateChange:
		problem = data.(Problem)
		if _, ok := s.problems[problem.ID]; !ok {
			return nil, fmt.Errorf(
				"problem with id = %d does not exists", problem.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET "owner_id" = $2 WHERE "id" = $1"`,
				s.table,
			),
			problem.ID, problem.OwnerID,
		)
		if err != nil {
			return nil, err
		}
	case DeleteChange:
		var ok bool
		problem, ok = s.problems[data.(int64)]
		if !ok {
			return nil, fmt.Errorf(
				"problem with id = %d does not exists", problem.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1"`,
				s.table,
			),
			problem.ID,
		)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf(
			"unsupported change type = %d", changeType,
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
		changeType, changeTime, problem.ID,
		problem.OwnerID, problem.CreateTime,
	)
	if err != nil {
		return nil, err
	}
	changeID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &ProblemChange{
		ChangeBase: ChangeBase{
			ID: changeID, Type: changeType, Time: changeTime,
		},
		Problem: problem,
	}, nil
}

func (s *ProblemStore) applyChange(change Change) {
	problem := change.ChangeData().(Problem)
	switch change.ChangeType() {
	case CreateChange:
		s.problems[problem.ID] = problem
	case UpdateChange:
		s.problems[problem.ID] = problem
	case DeleteChange:
		delete(s.problems, problem.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %d", change.ChangeType(),
		))
	}
}

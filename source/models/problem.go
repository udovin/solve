package models

import (
	"database/sql"
	"fmt"
	"time"
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

func (s *ProblemStore) Get(id int64) (Problem, bool) {
	problem, ok := s.problems[id]
	return problem, ok
}

func (s *ProblemStore) Create(m *Problem) error {
	change := ProblemChange{
		ChangeBase: ChangeBase{Type: CreateChange},
		Problem:    *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Problem
	return nil
}

func (s *ProblemStore) Update(m *Problem) error {
	change := ProblemChange{
		ChangeBase: ChangeBase{Type: UpdateChange},
		Problem:    *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Problem
	return nil
}

func (s *ProblemStore) Delete(id int64) error {
	change := ProblemChange{
		ChangeBase: ChangeBase{Type: DeleteChange},
		Problem:    Problem{ID: id},
	}
	return s.Manager.Change(&change)
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

func (s *ProblemStore) saveChangeTx(tx *ChangeTx, change Change) error {
	problem := change.(*ProblemChange)
	problem.Time = time.Now().Unix()
	switch problem.Type {
	case CreateChange:
		problem.Problem.CreateTime = problem.Time
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
			return err
		}
		problem.Problem.ID, err = res.LastInsertId()
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.problems[problem.Problem.ID]; !ok {
			return fmt.Errorf(
				"problem with id = %d does not exists",
				problem.Problem.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET "owner_id" = $1 WHERE "id" = $2`,
				s.table,
			),
			problem.OwnerID, problem.Problem.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.problems[problem.Problem.ID]; !ok {
			return fmt.Errorf(
				"problem with id = %d does not exists",
				problem.Problem.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1`,
				s.table,
			),
			problem.Problem.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			problem.Type,
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
		problem.Type, problem.Time,
		problem.Problem.ID, problem.OwnerID, problem.CreateTime,
	)
	if err != nil {
		return err
	}
	problem.ChangeBase.ID, err = res.LastInsertId()
	return err
}

func (s *ProblemStore) applyChange(change Change) {
	problemChange := change.(*ProblemChange)
	problem := problemChange.Problem
	switch problemChange.Type {
	case CreateChange:
		s.problems[problem.ID] = problem
	case UpdateChange:
		s.problems[problem.ID] = problem
	case DeleteChange:
		delete(s.problems, problem.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			problemChange.Type,
		))
	}
}

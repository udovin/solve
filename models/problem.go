package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Problem struct {
	ID         int64 `json:"" db:"id"`
	UserID     int64 `json:"" db:"user_id"`
	CreateTime int64 `json:"" db:"create_time"`
}

type ProblemChange struct {
	BaseChange
	Problem
}

type ProblemStore struct {
	Manager     *ChangeManager
	table       string
	changeTable string
	problems    map[int64]Problem
	mutex       sync.RWMutex
}

func NewProblemStore(db *sql.DB, table, changeTable string) *ProblemStore {
	store := ProblemStore{
		table:       table,
		changeTable: changeTable,
		problems:    make(map[int64]Problem),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *ProblemStore) Get(id int64) (Problem, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if problem, ok := s.problems[id]; ok {
		return problem, nil
	}
	return Problem{}, sql.ErrNoRows
}

func (s *ProblemStore) Create(m *Problem) error {
	change := ProblemChange{
		BaseChange: BaseChange{Type: CreateChange},
		Problem:    *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Problem
	return nil
}

func (s *ProblemStore) CreateTx(tx *ChangeTx, m *Problem) error {
	change := ProblemChange{
		BaseChange: BaseChange{Type: CreateChange},
		Problem:    *m,
	}
	err := s.Manager.ChangeTx(tx, &change)
	if err != nil {
		return err
	}
	*m = change.Problem
	return nil
}

func (s *ProblemStore) Update(m *Problem) error {
	change := ProblemChange{
		BaseChange: BaseChange{Type: UpdateChange},
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
		BaseChange: BaseChange{Type: DeleteChange},
		Problem:    Problem{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *ProblemStore) GetLocker() sync.Locker {
	return &s.mutex
}

func (s *ProblemStore) InitChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *ProblemStore) LoadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "user_id", "create_time"`+
				` FROM %q`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *ProblemStore) ScanChange(scan Scanner) (Change, error) {
	problem := ProblemChange{}
	err := scan.Scan(
		&problem.BaseChange.ID, &problem.Type, &problem.Time,
		&problem.Problem.ID, &problem.UserID, &problem.CreateTime,
	)
	return &problem, err
}

func (s *ProblemStore) SaveChange(tx *sql.Tx, change Change) error {
	problem := change.(*ProblemChange)
	problem.Time = time.Now().Unix()
	switch problem.Type {
	case CreateChange:
		problem.Problem.CreateTime = problem.Time
		var err error
		problem.Problem.ID, err = execTxReturningID(
			s.Manager.db.Driver(), tx,
			fmt.Sprintf(
				`INSERT INTO %q`+
					` ("user_id", "create_time")`+
					` VALUES ($1, $2)`,
				s.table,
			),
			"id",
			problem.UserID, problem.CreateTime,
		)
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
				`UPDATE %q SET "user_id" = $1 WHERE "id" = $2`,
				s.table,
			),
			problem.UserID, problem.Problem.ID,
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
				`DELETE FROM %q WHERE "id" = $1`,
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
	var err error
	problem.BaseChange.ID, err = execTxReturningID(
		s.Manager.db.Driver(), tx,
		fmt.Sprintf(
			`INSERT INTO %q`+
				` ("change_type", "change_time",`+
				` "id", "user_id", "create_time")`+
				` VALUES ($1, $2, $3, $4, $5)`,
			s.changeTable,
		),
		"change_id",
		problem.Type, problem.Time,
		problem.Problem.ID, problem.UserID, problem.CreateTime,
	)
	return err
}

func (s *ProblemStore) ApplyChange(change Change) {
	problem := change.(*ProblemChange)
	switch problem.Type {
	case UpdateChange:
		fallthrough
	case CreateChange:
		s.problems[problem.Problem.ID] = problem.Problem
	case DeleteChange:
		delete(s.problems, problem.Problem.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			problem.Type,
		))
	}
}

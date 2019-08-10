package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Problem struct {
	ID         int64 `json:"" db:"id"`
	OwnerID    int64 `json:"" db:"owner_id"`
	CreateTime int64 `json:"" db:"create_time"`
}

type problemChange struct {
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

func (s *ProblemStore) Get(id int64) (Problem, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	problem, ok := s.problems[id]
	return problem, ok
}

func (s *ProblemStore) Create(m *Problem) error {
	change := problemChange{
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

func (s *ProblemStore) Update(m *Problem) error {
	change := problemChange{
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
	change := problemChange{
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
				` "id", "owner_id", "create_time"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *ProblemStore) ScanChange(scan Scanner) (Change, error) {
	problem := problemChange{}
	err := scan.Scan(
		&problem.BaseChange.ID, &problem.Type, &problem.Time,
		&problem.Problem.ID, &problem.OwnerID, &problem.CreateTime,
	)
	return &problem, err
}

func (s *ProblemStore) SaveChange(tx *sql.Tx, change Change) error {
	problem := change.(*problemChange)
	problem.Time = time.Now().Unix()
	switch problem.Type {
	case CreateChange:
		problem.Problem.CreateTime = problem.Time
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("owner_id", "create_time")`+
					` VALUES ($1, $2)`,
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
			`INSERT INTO "%s"`+
				` ("change_type", "change_time",`+
				` "id", "owner_id", "create_time")`+
				` VALUES ($1, $2, $3, $4, $5)`,
			s.changeTable,
		),
		problem.Type, problem.Time,
		problem.Problem.ID, problem.OwnerID, problem.CreateTime,
	)
	if err != nil {
		return err
	}
	problem.BaseChange.ID, err = res.LastInsertId()
	return err
}

func (s *ProblemStore) ApplyChange(change Change) {
	problem := change.(*problemChange)
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

package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Solution struct {
	ID         int64  `json:"" db:"id"`
	UserID     int64  `json:"" db:"user_id"`
	ProblemID  int64  `json:"" db:"problem_id"`
	CompilerID int64  `json:"" db:"compiler_id"`
	SourceCode string `json:"" db:"source_code"`
	CreateTime int64  `json:"" db:"create_time"`
}

type solutionChange struct {
	BaseChange
	Solution
}

type SolutionStore struct {
	Manager     *ChangeManager
	table       string
	changeTable string
	solutions   map[int64]Solution
	mutex       sync.RWMutex
}

func NewSolutionStore(db *sql.DB, table, changeTable string) *SolutionStore {
	store := SolutionStore{
		table:       table,
		changeTable: changeTable,
		solutions:   make(map[int64]Solution),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *SolutionStore) Get(id int64) (Solution, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	solution, ok := s.solutions[id]
	return solution, ok
}

func (s *SolutionStore) Create(m *Solution) error {
	change := solutionChange{
		BaseChange: BaseChange{Type: CreateChange},
		Solution:   *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Solution
	return nil
}

func (s *SolutionStore) Update(m *Solution) error {
	change := solutionChange{
		BaseChange: BaseChange{Type: UpdateChange},
		Solution:   *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Solution
	return nil
}

func (s *SolutionStore) Delete(id int64) error {
	change := solutionChange{
		BaseChange: BaseChange{Type: DeleteChange},
		Solution:   Solution{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *SolutionStore) GetLocker() sync.Locker {
	return &s.mutex
}

func (s *SolutionStore) InitChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *SolutionStore) LoadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "user_id", "problem_id", "compiler_id",`+
				` "source_code", "create_time"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *SolutionStore) ScanChange(scan Scanner) (Change, error) {
	solution := solutionChange{}
	err := scan.Scan(
		&solution.BaseChange.ID, &solution.Type, &solution.Time,
		&solution.Solution.ID, &solution.UserID, &solution.ProblemID,
		&solution.CompilerID, &solution.SourceCode, &solution.CreateTime,
	)
	return &solution, err
}

func (s *SolutionStore) SaveChange(tx *sql.Tx, change Change) error {
	solution := change.(*solutionChange)
	solution.Time = time.Now().Unix()
	switch solution.Type {
	case CreateChange:
		solution.Solution.CreateTime = solution.Time
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("user_id", "problem_id", "compiler_id",`+
					` "source_code", "create_time")`+
					` VALUES ($1, $2, $3, $4, $5)`,
				s.table,
			),
			solution.UserID, solution.ProblemID, solution.CompilerID,
			solution.SourceCode, solution.CreateTime,
		)
		if err != nil {
			return err
		}
		solution.Solution.ID, err = res.LastInsertId()
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.solutions[solution.Solution.ID]; !ok {
			return fmt.Errorf(
				"solution with id = %d does not exists",
				solution.Solution.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s"`+
					` SET "user_id" = $1, "problem_id" = $2,`+
					` "compiler_id" = $3, "source_code" = $4`+
					` WHERE "id" = $5`,
				s.table,
			),
			solution.UserID, solution.ProblemID, solution.CompilerID,
			solution.SourceCode, solution.Solution.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.solutions[solution.Solution.ID]; !ok {
			return fmt.Errorf(
				"solution with id = %d does not exists",
				solution.Solution.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1`,
				s.table,
			),
			solution.Solution.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			solution.Type,
		)
	}
	res, err := tx.Exec(
		fmt.Sprintf(
			`INSERT INTO "%s"`+
				` ("change_type", "change_time",`+
				` "id", "user_id", "problem_id", "compiler_id",`+
				` "source_code", "create_time")`+
				` VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			s.changeTable,
		),
		solution.Type, solution.Time,
		solution.Solution.ID, solution.UserID, solution.ProblemID,
		solution.CompilerID, solution.SourceCode,
		solution.CreateTime,
	)
	if err != nil {
		return err
	}
	solution.BaseChange.ID, err = res.LastInsertId()
	return err
}

func (s *SolutionStore) ApplyChange(change Change) {
	solution := change.(*solutionChange)
	switch solution.Type {
	case UpdateChange:
		fallthrough
	case CreateChange:
		s.solutions[solution.Solution.ID] = solution.Solution
	case DeleteChange:
		delete(s.solutions, solution.Solution.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			solution.Type,
		))
	}
}

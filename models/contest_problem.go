package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type ContestProblem struct {
	ContestID int64  `json:"" db:"contest_id"`
	ProblemID int64  `json:"" db:"problem_id"`
	Code      string `json:"" db:"code"`
}

type contestProblemChange struct {
	BaseChange
	ContestProblem
}

type ContestProblemStore struct {
	Manager     *ChangeManager
	table       string
	changeTable string
	problems    map[int64]map[int64]ContestProblem
	mutex       sync.RWMutex
}

func NewContestProblemStore(
	db *sql.DB, table, changeTable string,
) *ContestProblemStore {
	store := ContestProblemStore{
		table:       table,
		changeTable: changeTable,
		problems:    make(map[int64]map[int64]ContestProblem),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *ContestProblemStore) GetByContest(
	id int64,
) ([]ContestProblem, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var found []ContestProblem
	if problems, ok := s.problems[id]; ok {
		for _, problem := range problems {
			found = append(found, problem)
		}
	}
	return found, nil
}

func (s *ContestProblemStore) Create(m *ContestProblem) error {
	change := contestProblemChange{
		BaseChange:     BaseChange{Type: CreateChange},
		ContestProblem: *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.ContestProblem
	return nil
}

func (s *ContestProblemStore) Update(m *ContestProblem) error {
	change := contestProblemChange{
		BaseChange:     BaseChange{Type: UpdateChange},
		ContestProblem: *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.ContestProblem
	return nil
}

func (s *ContestProblemStore) Delete(contestID int64, problemID int64) error {
	change := contestProblemChange{
		BaseChange: BaseChange{Type: DeleteChange},
		ContestProblem: ContestProblem{
			ContestID: contestID,
			ProblemID: problemID,
		},
	}
	return s.Manager.Change(&change)
}

func (s *ContestProblemStore) GetLocker() sync.Locker {
	return &s.mutex
}

func (s *ContestProblemStore) InitChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *ContestProblemStore) LoadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "contest_id", "problem_id", "code"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *ContestProblemStore) ScanChange(scan Scanner) (Change, error) {
	problem := contestProblemChange{}
	err := scan.Scan(
		&problem.BaseChange.ID, &problem.Type, &problem.Time,
		&problem.ContestID, &problem.ProblemID, &problem.Code,
	)
	return &problem, err
}

func (s *ContestProblemStore) SaveChange(tx *sql.Tx, change Change) error {
	problem := change.(*contestProblemChange)
	problem.Time = time.Now().Unix()
	switch problem.Type {
	case CreateChange:
		_, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("contest_id", "problem_id", "code")`+
					` VALUES ($1, $2, $3)`,
				s.table,
			),
			problem.ContestID, problem.ProblemID, problem.Code,
		)
		if err != nil {
			return err
		}
	case UpdateChange:
		problems, ok := s.problems[problem.ContestID]
		if !ok {
			return fmt.Errorf(
				"problem with contest ID = %d does not exists",
				problem.ContestID,
			)
		}
		if _, ok := problems[problem.ProblemID]; !ok {
			return fmt.Errorf(
				"problem with problem ID = %d does not exists",
				problem.ProblemID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET "code" = $1`+
					` WHERE "contest_id" = $2 AND "problem_id" = $3`,
				s.table,
			),
			problem.Code, problem.ContestID, problem.ProblemID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		problems, ok := s.problems[problem.ContestID]
		if !ok {
			return fmt.Errorf(
				"problem with contest ID = %d does not exists",
				problem.ContestID,
			)
		}
		if _, ok := problems[problem.ProblemID]; !ok {
			return fmt.Errorf(
				"problem with problem ID = %d does not exists",
				problem.ProblemID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s"`+
					` WHERE "contest_id" = $1 AND "problem_id" = $2`,
				s.table,
			),
			problem.ContestID, problem.ProblemID,
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
			`INSERT INTO "%s"`+
				` ("change_type", "change_time",`+
				` "contest_id", "problem_id", "code")`+
				` VALUES ($1, $2, $3, $4, $5)`,
			s.changeTable,
		),
		"change_id",
		problem.Type, problem.Time,
		problem.ContestID, problem.ProblemID, problem.Code,
	)
	return err
}

func (s *ContestProblemStore) ApplyChange(change Change) {
	problem := change.(*contestProblemChange)
	switch problem.Type {
	case UpdateChange:
		fallthrough
	case CreateChange:
		if _, ok := s.problems[problem.ContestID]; !ok {
			s.problems[problem.ContestID] = make(map[int64]ContestProblem)
		}
		s.problems[problem.ContestID][problem.ProblemID] =
			problem.ContestProblem
	case DeleteChange:
		if problems, ok := s.problems[problem.ContestID]; ok {
			delete(problems, problem.ProblemID)
			if len(problems) == 0 {
				delete(s.problems, problem.ContestID)
			}
		}
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			problem.Type,
		))
	}
}

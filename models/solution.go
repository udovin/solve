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
	ContestID  int64  `json:"" db:"contest_id"`
	CompilerID int64  `json:"" db:"compiler_id"`
	SourceCode string `json:"" db:"source_code"`
	CreateTime int64  `json:"" db:"create_time"`
}

type solutionChange struct {
	BaseChange
	Solution
}

type problemUserPair struct {
	ProblemID int64
	UserID    int64
}

type SolutionStore struct {
	Manager          *ChangeManager
	table            string
	changeTable      string
	solutions        map[int64]Solution
	contestSolutions map[int64]map[int64]struct{}
	problemUser      map[problemUserPair]map[int64]struct{}
	mutex            sync.RWMutex
}

func NewSolutionStore(db *sql.DB, table, changeTable string) *SolutionStore {
	store := SolutionStore{
		table:            table,
		changeTable:      changeTable,
		solutions:        make(map[int64]Solution),
		contestSolutions: make(map[int64]map[int64]struct{}),
		problemUser:      make(map[problemUserPair]map[int64]struct{}),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *SolutionStore) Get(id int64) (Solution, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if solution, ok := s.solutions[id]; ok {
		return solution, nil
	}
	return Solution{}, sql.ErrNoRows
}

func (s *SolutionStore) All() ([]Solution, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var solutions []Solution
	for _, solution := range s.solutions {
		solutions = append(solutions, solution)
	}
	return solutions, nil
}

func (s *SolutionStore) GetByContest(contestID int64) ([]Solution, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var solutions []Solution
	if contest, ok := s.contestSolutions[contestID]; ok {
		for id := range contest {
			if solution, ok := s.solutions[id]; ok {
				solutions = append(solutions, solution)
			}
		}
	}
	return solutions, nil
}

func (s *SolutionStore) GetByProblemUser(
	problemID, userID int64,
) ([]Solution, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	key := problemUserPair{
		ProblemID: problemID,
		UserID:    userID,
	}
	var solutions []Solution
	if ids, ok := s.problemUser[key]; ok {
		for id := range ids {
			if solution, ok := s.solutions[id]; ok {
				solutions = append(solutions, solution)
			}
		}
	}
	return solutions, nil
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

func (s *SolutionStore) CreateTx(tx *ChangeTx, m *Solution) error {
	change := solutionChange{
		BaseChange: BaseChange{Type: CreateChange},
		Solution:   *m,
	}
	err := s.Manager.ChangeTx(tx, &change)
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
				` "id", "user_id", "problem_id", "contest_id",`+
				` "compiler_id", "source_code", "create_time"`+
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
	var contestID *int64
	err := scan.Scan(
		&solution.BaseChange.ID, &solution.Type, &solution.Time,
		&solution.Solution.ID, &solution.UserID, &solution.ProblemID,
		&contestID, &solution.CompilerID, &solution.SourceCode,
		&solution.CreateTime,
	)
	if contestID != nil {
		solution.ContestID = *contestID
	}
	return &solution, err
}

func int64OrNil(i int64) *int64 {
	if i == 0 {
		return nil
	}
	return &i
}

func (s *SolutionStore) SaveChange(tx *sql.Tx, change Change) error {
	solution := change.(*solutionChange)
	solution.Time = time.Now().Unix()
	switch solution.Type {
	case CreateChange:
		solution.Solution.CreateTime = solution.Time
		var err error
		solution.Solution.ID, err = execTxReturningID(
			s.Manager.db.Driver(), tx,
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("user_id", "problem_id", "contest_id", "compiler_id",`+
					` "source_code", "create_time")`+
					` VALUES ($1, $2, $3, $4, $5, $6)`,
				s.table,
			),
			"id",
			solution.UserID, solution.ProblemID,
			int64OrNil(solution.ContestID), solution.CompilerID,
			solution.SourceCode, solution.CreateTime,
		)
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
					` "contest_id" = $3, "compiler_id" = $4,`+
					` "source_code" = $5`+
					` WHERE "id" = $6`,
				s.table,
			),
			solution.UserID, solution.ProblemID,
			int64OrNil(solution.ContestID), solution.CompilerID,
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
	var err error
	solution.BaseChange.ID, err = execTxReturningID(
		s.Manager.db.Driver(), tx,
		fmt.Sprintf(
			`INSERT INTO "%s"`+
				` ("change_type", "change_time",`+
				` "id", "user_id", "problem_id", "contest_id",`+
				` "compiler_id", "source_code", "create_time")`+
				` VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			s.changeTable,
		),
		"change_id",
		solution.Type, solution.Time,
		solution.Solution.ID, solution.UserID, solution.ProblemID,
		int64OrNil(solution.ContestID), solution.CompilerID,
		solution.SourceCode, solution.CreateTime,
	)
	return err
}

func (s *SolutionStore) ApplyChange(change Change) {
	solution := change.(*solutionChange)
	problemUser := problemUserPair{
		ProblemID: solution.ProblemID,
		UserID:    solution.UserID,
	}
	switch solution.Type {
	case UpdateChange:
		if old, ok := s.solutions[solution.Solution.ID]; ok {
			oldKey := problemUserPair{
				ProblemID: old.ProblemID,
				UserID:    old.UserID,
			}
			if oldKey != problemUser {
				if solution, ok := s.problemUser[oldKey]; ok {
					delete(solution, old.ID)
					if len(solution) == 0 {
						delete(s.problemUser, oldKey)
					}
				}
			}
			if old.ContestID != solution.ContestID {
				if fields, ok := s.contestSolutions[old.ContestID]; ok {
					delete(fields, old.ID)
					if len(fields) == 0 {
						delete(s.contestSolutions, old.ContestID)
					}
				}
			}
		}
		fallthrough
	case CreateChange:
		s.solutions[solution.Solution.ID] = solution.Solution
		if _, ok := s.problemUser[problemUser]; !ok {
			s.problemUser[problemUser] = make(map[int64]struct{})
		}
		s.problemUser[problemUser][solution.Solution.ID] = struct{}{}
		if _, ok := s.contestSolutions[solution.ContestID]; !ok {
			s.contestSolutions[solution.ContestID] = make(map[int64]struct{})
		}
		s.contestSolutions[solution.ContestID][solution.Solution.ID] = struct{}{}
	case DeleteChange:
		delete(s.solutions, solution.Solution.ID)
		if fields, ok := s.problemUser[problemUser]; ok {
			delete(fields, solution.Solution.ID)
			if len(fields) == 0 {
				delete(s.problemUser, problemUser)
			}
		}
		if fields, ok := s.contestSolutions[solution.ContestID]; ok {
			delete(fields, solution.Solution.ID)
			if len(fields) == 0 {
				delete(s.contestSolutions, solution.ContestID)
			}
		}
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			solution.Type,
		))
	}
}

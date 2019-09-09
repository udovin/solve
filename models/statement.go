package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Statement struct {
	ID          int64  `json:"" db:"id"`
	ProblemID   int64  `json:"" db:"problem_id"`
	Title       string `json:"" db:"title"`
	Description string `json:"" db:"description"`
	CreateTime  int64  `json:"" db:"create_time"`
}

type statementChange struct {
	BaseChange
	Statement
}

type StatementStore struct {
	Manager           *ChangeManager
	table             string
	changeTable       string
	statements        map[int64]Statement
	problemStatements map[int64]int64
	mutex             sync.RWMutex
}

func NewStatementStore(db *sql.DB, table, changeTable string) *StatementStore {
	store := StatementStore{
		table:             table,
		changeTable:       changeTable,
		statements:        make(map[int64]Statement),
		problemStatements: make(map[int64]int64),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *StatementStore) GetByProblem(id int64) (Statement, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if statementID, ok := s.problemStatements[id]; ok {
		if statement, ok := s.statements[statementID]; ok {
			return statement, true
		}
	}
	return Statement{}, false
}

func (s *StatementStore) Create(m *Statement) error {
	change := statementChange{
		BaseChange: BaseChange{Type: CreateChange},
		Statement:  *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Statement
	return nil
}

func (s *StatementStore) CreateTx(tx *ChangeTx, m *Statement) error {
	change := statementChange{
		BaseChange: BaseChange{Type: CreateChange},
		Statement:  *m,
	}
	err := s.Manager.ChangeTx(tx, &change)
	if err != nil {
		return err
	}
	*m = change.Statement
	return nil
}

func (s *StatementStore) Update(m *Statement) error {
	change := statementChange{
		BaseChange: BaseChange{Type: UpdateChange},
		Statement:  *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Statement
	return nil
}

func (s *StatementStore) Delete(id int64) error {
	change := statementChange{
		BaseChange: BaseChange{Type: DeleteChange},
		Statement:  Statement{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *StatementStore) GetLocker() sync.Locker {
	return &s.mutex
}

func (s *StatementStore) InitChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *StatementStore) LoadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "problem_id", "title", "description",`+
				` "create_time"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *StatementStore) ScanChange(scan Scanner) (Change, error) {
	statement := statementChange{}
	err := scan.Scan(
		&statement.BaseChange.ID, &statement.Type, &statement.Time,
		&statement.Statement.ID, &statement.ProblemID, &statement.Title,
		&statement.Description, &statement.CreateTime,
	)
	return &statement, err
}

func (s *StatementStore) SaveChange(tx *sql.Tx, change Change) error {
	statement := change.(*statementChange)
	statement.Time = time.Now().Unix()
	switch statement.Type {
	case CreateChange:
		statement.Statement.CreateTime = statement.Time
		var err error
		statement.Statement.ID, err = execTxReturningID(
			s.Manager.db.Driver(), tx,
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("problem_id", "title", "description",`+
					` "create_time")`+
					` VALUES ($1, $2, $3, $4)`,
				s.table,
			),
			"id",
			statement.ProblemID, statement.Title, statement.Description,
			statement.CreateTime,
		)
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.statements[statement.Statement.ID]; !ok {
			return fmt.Errorf(
				"statement with id = %d does not exists",
				statement.Statement.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s"`+
					` SET "problem_id" = $1, "title" = $2,`+
					` "description" = $3`+
					` WHERE "id" = $4`,
				s.table,
			),
			statement.ProblemID, statement.Title, statement.Description,
			statement.Statement.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.statements[statement.Statement.ID]; !ok {
			return fmt.Errorf(
				"statement with id = %d does not exists",
				statement.Statement.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1`,
				s.table,
			),
			statement.Statement.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			statement.Type,
		)
	}
	var err error
	statement.BaseChange.ID, err = execTxReturningID(
		s.Manager.db.Driver(), tx,
		fmt.Sprintf(
			`INSERT INTO "%s"`+
				` ("change_type", "change_time",`+
				` "id", "problem_id", "title", "description",`+
				` "create_time")`+
				` VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			s.changeTable,
		),
		"change_id",
		statement.Type, statement.Time,
		statement.Statement.ID, statement.ProblemID, statement.Title,
		statement.Description, statement.CreateTime,
	)
	return err
}

func (s *StatementStore) ApplyChange(change Change) {
	statement := change.(*statementChange)
	switch statement.Type {
	case UpdateChange:
		fallthrough
	case CreateChange:
		s.statements[statement.Statement.ID] = statement.Statement
		s.problemStatements[statement.ProblemID] = statement.Statement.ID
	case DeleteChange:
		delete(s.problemStatements, statement.ProblemID)
		delete(s.statements, statement.Statement.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			statement.Type,
		))
	}
}

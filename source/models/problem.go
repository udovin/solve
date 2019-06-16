package models

import (
	"database/sql"
	"fmt"
)

type Problem struct {
	ID         int64 `db:"id"          json:""`
	CreateTime int64 `db:"create_time" json:""`
}

type ProblemChange struct {
	Problem
	ID   int64      `db:"change_id"   json:""`
	Type ChangeType `db:"change_type" json:""`
}

type ProblemStore struct {
	db          *sql.DB
	table       string
	changeTable string
	problems    map[int64]Problem
}

func (c *ProblemChange) ChangeID() int64 {
	return c.ID
}

func (c *ProblemChange) ChangeType() ChangeType {
	return c.Type
}

func (c *ProblemChange) ChangeData() interface{} {
	return c.Problem
}

func NewProblemStore(
	db *sql.DB, table, changeTable string,
) *ProblemStore {
	return &ProblemStore{
		db:          db,
		table:       table,
		changeTable: changeTable,
	}
}

func (s *ProblemStore) GetDB() *sql.DB {
	return s.db
}

func (s *ProblemStore) TableName() string {
	return s.table
}

func (s *ProblemStore) ChangeTableName() string {
	return s.changeTable
}

func (s *ProblemStore) scanChange(scan RowScan) (Change, error) {
	change := &ProblemChange{}
	if err := scan.Scan(change); err != nil {
		return nil, err
	}
	return change, nil
}

func (s *ProblemStore) applyChange(change Change) error {
	problem := change.ChangeData().(Problem)
	switch change.ChangeType() {
	case CreateChange:
		s.problems[problem.ID] = problem
	case UpdateChange:
		s.problems[problem.ID] = problem
	case DeleteChange:
		delete(s.problems, problem.ID)
	default:
		return fmt.Errorf(
			"unsupported change type = %d", change.ChangeType(),
		)
	}
	return nil
}

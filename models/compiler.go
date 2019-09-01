package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Compiler struct {
	ID         int64  `json:"" db:"id"`
	Name       string `json:"" db:"name"`
	CreateTime int64  `json:"" db:"create_time"`
}

type compilerChange struct {
	BaseChange
	Compiler
}

type CompilerStore struct {
	Manager     *ChangeManager
	table       string
	changeTable string
	compilers   map[int64]Compiler
	mutex       sync.RWMutex
}

func NewCompilerStore(db *sql.DB, table, changeTable string) *CompilerStore {
	store := CompilerStore{
		table:       table,
		changeTable: changeTable,
		compilers:   make(map[int64]Compiler),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *CompilerStore) Get(id int64) (Compiler, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	compiler, ok := s.compilers[id]
	return compiler, ok
}

func (s *CompilerStore) Create(m *Compiler) error {
	change := compilerChange{
		BaseChange: BaseChange{Type: CreateChange},
		Compiler:   *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Compiler
	return nil
}

func (s *CompilerStore) Update(m *Compiler) error {
	change := compilerChange{
		BaseChange: BaseChange{Type: UpdateChange},
		Compiler:   *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Compiler
	return nil
}

func (s *CompilerStore) Delete(id int64) error {
	change := compilerChange{
		BaseChange: BaseChange{Type: DeleteChange},
		Compiler:   Compiler{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *CompilerStore) GetLocker() sync.Locker {
	return &s.mutex
}

func (s *CompilerStore) InitChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *CompilerStore) LoadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "name", "create_time"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *CompilerStore) ScanChange(scan Scanner) (Change, error) {
	compiler := compilerChange{}
	err := scan.Scan(
		&compiler.BaseChange.ID, &compiler.Type, &compiler.Time,
		&compiler.Compiler.ID, &compiler.Name, &compiler.CreateTime,
	)
	return &compiler, err
}

func (s *CompilerStore) SaveChange(tx *sql.Tx, change Change) error {
	compiler := change.(*compilerChange)
	compiler.Time = time.Now().Unix()
	switch compiler.Type {
	case CreateChange:
		compiler.Compiler.CreateTime = compiler.Time
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("name", "create_time")`+
					` VALUES ($1, $2)`,
				s.table,
			),
			compiler.Name, compiler.CreateTime,
		)
		if err != nil {
			return err
		}
		compiler.Compiler.ID, err = res.LastInsertId()
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.compilers[compiler.Compiler.ID]; !ok {
			return fmt.Errorf(
				"compiler with id = %d does not exists",
				compiler.Compiler.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET "name" = $1 WHERE "id" = $2`,
				s.table,
			),
			compiler.Name, compiler.Compiler.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.compilers[compiler.Compiler.ID]; !ok {
			return fmt.Errorf(
				"compiler with id = %d does not exists",
				compiler.Compiler.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1`,
				s.table,
			),
			compiler.Compiler.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			compiler.Type,
		)
	}
	res, err := tx.Exec(
		fmt.Sprintf(
			`INSERT INTO "%s"`+
				` ("change_type", "change_time",`+
				` "id", "name", "create_time")`+
				` VALUES ($1, $2, $3, $4, $5)`,
			s.changeTable,
		),
		compiler.Type, compiler.Time,
		compiler.Compiler.ID, compiler.Name, compiler.CreateTime,
	)
	if err != nil {
		return err
	}
	compiler.BaseChange.ID, err = res.LastInsertId()
	return err
}

func (s *CompilerStore) ApplyChange(change Change) {
	compiler := change.(*compilerChange)
	switch compiler.Type {
	case UpdateChange:
		fallthrough
	case CreateChange:
		s.compilers[compiler.Compiler.ID] = compiler.Compiler
	case DeleteChange:
		delete(s.compilers, compiler.Compiler.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			compiler.Type,
		))
	}
}

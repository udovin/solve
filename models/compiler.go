package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// Compiler contains information about compiler
type Compiler struct {
	ID         int64  `json:"" db:"id"`
	Name       string `json:"" db:"name"`
	CreateTime int64  `json:"" db:"create_time"`
}

// CompilerChange represents a change of compiler
type CompilerChange struct {
	BaseChange
	Compiler
}

// CompilerStore manages all compiler models
type CompilerStore struct {
	Manager     *ChangeManager
	table       string
	changeTable string
	compilers   map[int64]Compiler
	mutex       sync.RWMutex
}

// NewCompilerStore creates a new instance of CompilerStore
func NewCompilerStore(db *sql.DB, table, changeTable string) *CompilerStore {
	store := CompilerStore{
		table:       table,
		changeTable: changeTable,
		compilers:   make(map[int64]Compiler),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

// All returns all compilers
func (s *CompilerStore) All() ([]Compiler, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var compilers []Compiler
	for _, compiler := range s.compilers {
		compilers = append(compilers, compiler)
	}
	return compilers, nil
}

// Get returns compiler by its ID in the store
func (s *CompilerStore) Get(id int64) (Compiler, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if compiler, ok := s.compilers[id]; ok {
		return compiler, nil
	}
	return Compiler{}, sql.ErrNoRows
}

// Create creates new compiler in the store
func (s *CompilerStore) Create(m *Compiler) error {
	change := CompilerChange{
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

// Update persists update of compiler to store
func (s *CompilerStore) Update(m *Compiler) error {
	change := CompilerChange{
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

// Delete deletes compiler from store by its ID
func (s *CompilerStore) Delete(id int64) error {
	change := CompilerChange{
		BaseChange: BaseChange{Type: DeleteChange},
		Compiler:   Compiler{ID: id},
	}
	return s.Manager.Change(&change)
}

// GetLocker returns locker that should be
// acquired by manager when store will be changed
func (s *CompilerStore) GetLocker() sync.Locker {
	return &s.mutex
}

// InitChanges initializes store with initial state
func (s *CompilerStore) InitChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

// LoadChanges loads changes from store
func (s *CompilerStore) LoadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "name", "create_time"`+
				` FROM %q`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

// ScanChange scans change struct using Scanner interface
func (s *CompilerStore) ScanChange(scan Scanner) (Change, error) {
	compiler := CompilerChange{}
	err := scan.Scan(
		&compiler.BaseChange.ID, &compiler.Type, &compiler.Time,
		&compiler.Compiler.ID, &compiler.Name, &compiler.CreateTime,
	)
	return &compiler, err
}

// SaveChange persists change to database in transaction
func (s *CompilerStore) SaveChange(tx *sql.Tx, change Change) error {
	compiler := change.(*CompilerChange)
	compiler.Time = time.Now().Unix()
	switch compiler.Type {
	case CreateChange:
		compiler.Compiler.CreateTime = compiler.Time
		var err error
		compiler.Compiler.ID, err = execTxReturningID(
			s.Manager.db.Driver(), tx,
			fmt.Sprintf(
				`INSERT INTO %q`+
					` ("name", "create_time")`+
					` VALUES ($1, $2)`,
				s.table,
			),
			"id",
			compiler.Name, compiler.CreateTime,
		)
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
				`UPDATE %q SET "name" = $1 WHERE "id" = $2`,
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
				`DELETE FROM %q WHERE "id" = $1`,
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
	var err error
	compiler.BaseChange.ID, err = execTxReturningID(
		s.Manager.db.Driver(), tx,
		fmt.Sprintf(
			`INSERT INTO %q`+
				` ("change_type", "change_time",`+
				` "id", "name", "create_time")`+
				` VALUES ($1, $2, $3, $4, $5)`,
			s.changeTable,
		),
		"change_id",
		compiler.Type, compiler.Time,
		compiler.Compiler.ID, compiler.Name, compiler.CreateTime,
	)
	return err
}

// ApplyChange applies persistent change to store
func (s *CompilerStore) ApplyChange(change Change) {
	compiler := change.(*CompilerChange)
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

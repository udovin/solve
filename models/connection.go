package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Connection struct {
	ID      int64 `json:"" db:"id"`
	LeftID  int64 `json:"" db:"left_id"`
	RightID int64 `json:"" db:"right_id"`
}

type ConnectionChange struct {
	BaseChange
	Connection
}

type ConnectionStore struct {
	Manager     *ChangeManager
	table       string
	changeTable string
	connections map[int64]Connection
	leftIDs     map[int64]map[int64]struct{}
	rightIDs    map[int64]map[int64]struct{}
	// mutex contains rw mutex
	mutex sync.RWMutex
}

func NewConnectionStore(
	db *sql.DB, table, changeTable string,
) *ConnectionStore {
	store := ConnectionStore{
		table:       table,
		changeTable: changeTable,
		connections: make(map[int64]Connection),
		leftIDs:     make(map[int64]map[int64]struct{}),
		rightIDs:    make(map[int64]map[int64]struct{}),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *ConnectionStore) GetByLeft(id int64) ([]Connection, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if ids, ok := s.leftIDs[id]; ok {
		var connections []Connection
		for id := range ids {
			if connection, ok := s.connections[id]; ok {
				connections = append(connections, connection)
			}
		}
		return connections, nil
	}
	return nil, nil
}

func (s *ConnectionStore) GetByRight(id int64) ([]Connection, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if ids, ok := s.rightIDs[id]; ok {
		var connections []Connection
		for id := range ids {
			if connection, ok := s.connections[id]; ok {
				connections = append(connections, connection)
			}
		}
		return connections, nil
	}
	return nil, nil
}

func (s *ConnectionStore) Create(m *Connection) error {
	change := ConnectionChange{
		BaseChange: BaseChange{Type: CreateChange},
		Connection: *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Connection
	return nil
}

func (s *ConnectionStore) CreateTx(tx *ChangeTx, m *Connection) error {
	change := ConnectionChange{
		BaseChange: BaseChange{Type: CreateChange},
		Connection: *m,
	}
	err := s.Manager.ChangeTx(tx, &change)
	if err != nil {
		return err
	}
	*m = change.Connection
	return nil
}

func (s *ConnectionStore) Update(m *Connection) error {
	change := ConnectionChange{
		BaseChange: BaseChange{Type: UpdateChange},
		Connection: *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Connection
	return nil
}

func (s *ConnectionStore) UpdateTx(tx *ChangeTx, m *Connection) error {
	change := ConnectionChange{
		BaseChange: BaseChange{Type: UpdateChange},
		Connection: *m,
	}
	err := s.Manager.ChangeTx(tx, &change)
	if err != nil {
		return err
	}
	*m = change.Connection
	return nil
}

func (s *ConnectionStore) Delete(id int64) error {
	change := ConnectionChange{
		BaseChange: BaseChange{Type: DeleteChange},
		Connection: Connection{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *ConnectionStore) DeleteTx(tx *ChangeTx, id int64) error {
	change := ConnectionChange{
		BaseChange: BaseChange{Type: DeleteChange},
		Connection: Connection{ID: id},
	}
	return s.Manager.ChangeTx(tx, &change)
}

func (s *ConnectionStore) GetLocker() sync.Locker {
	return &s.mutex
}

func (s *ConnectionStore) InitChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *ConnectionStore) LoadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "left_id", "right_id"`+
				` FROM %q`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *ConnectionStore) ScanChange(scan Scanner) (Change, error) {
	change := ConnectionChange{}
	err := scan.Scan(
		&change.BaseChange.ID, &change.BaseChange.Type, &change.Time,
		&change.Connection.ID, &change.LeftID, &change.RightID,
	)
	return &change, err
}

func (s *ConnectionStore) SaveChange(tx *sql.Tx, c Change) error {
	change := c.(*ConnectionChange)
	change.Time = time.Now().Unix()
	switch change.BaseChange.Type {
	case CreateChange:
		var err error
		change.Connection.ID, err = execTxReturningID(
			s.Manager.db.Driver(), tx,
			fmt.Sprintf(
				`INSERT INTO %q ("left_id", "right_id")`+
					` VALUES ($1, $2)`,
				s.table,
			),
			"id",
			change.LeftID, change.RightID,
		)
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.connections[change.Connection.ID]; !ok {
			return fmt.Errorf(
				"connection with id = %d does not exists",
				change.Connection.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE %q SET`+
					` "left_id" = $1, "right_id" = $2`+
					` WHERE "id" = $3`,
				s.table,
			),
			change.LeftID, change.RightID,
			change.Connection.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.connections[change.Connection.ID]; !ok {
			return fmt.Errorf(
				"connection with id = %d does not exists",
				change.Connection.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(`DELETE FROM %q WHERE "id" = $1`, s.table),
			change.Connection.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			change.BaseChange.Type,
		)
	}
	var err error
	change.BaseChange.ID, err = execTxReturningID(
		s.Manager.db.Driver(), tx,
		fmt.Sprintf(
			`INSERT INTO %q`+
				` ("change_type", "change_time",`+
				` "id", "left_id", "right_id")`+
				` VALUES ($1, $2, $3, $4, $5)`,
			s.changeTable,
		),
		"change_id",
		change.Type, change.Time,
		change.Connection.ID, change.LeftID, change.RightID,
	)
	return err
}

func (s *ConnectionStore) ApplyChange(c Change) {
	change := c.(*ConnectionChange)
	switch change.BaseChange.Type {
	case UpdateChange:
		if old, ok := s.connections[change.Connection.ID]; ok {
			if old.LeftID != change.LeftID {
				if ids, ok := s.leftIDs[old.LeftID]; ok {
					delete(ids, old.ID)
					if len(ids) == 0 {
						delete(s.leftIDs, old.LeftID)
					}
				}
			}
			if old.RightID != change.RightID {
				if ids, ok := s.rightIDs[old.RightID]; ok {
					delete(ids, old.ID)
					if len(ids) == 0 {
						delete(s.rightIDs, old.RightID)
					}
				}
			}
		}
		fallthrough
	case CreateChange:
		if _, ok := s.leftIDs[change.LeftID]; !ok {
			s.leftIDs[change.LeftID] = make(map[int64]struct{})
		}
		s.leftIDs[change.LeftID][change.Connection.ID] = struct{}{}
		if _, ok := s.rightIDs[change.RightID]; !ok {
			s.rightIDs[change.RightID] = make(map[int64]struct{})
		}
		s.rightIDs[change.RightID][change.Connection.ID] = struct{}{}
		s.connections[change.Connection.ID] = change.Connection
	case DeleteChange:
		if fields, ok := s.leftIDs[change.LeftID]; ok {
			delete(fields, change.Connection.ID)
			if len(fields) == 0 {
				delete(s.leftIDs, change.LeftID)
			}
		}
		if fields, ok := s.rightIDs[change.RightID]; ok {
			delete(fields, change.Connection.ID)
			if len(fields) == 0 {
				delete(s.rightIDs, change.RightID)
			}
		}
		delete(s.connections, change.Connection.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			change.BaseChange.Type,
		))
	}
}

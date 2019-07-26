package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Role struct {
	ID     int64  `json:"" db:"id"`
	UserID int64  `json:"" db:"user_id"`
	Code   string `json:"" db:"code"`
}

type roleChange struct {
	BaseChange
	Role
}

type RoleStore struct {
	Manager     *ChangeManager
	table       string
	changeTable string
	roles       map[int64]Role
	userRoles   map[int64]map[int64]struct{}
	mutex       sync.RWMutex
}

func (c *roleChange) ChangeData() interface{} {
	return c.Role
}

func NewRoleStore(db *sql.DB, table, changeTable string) *RoleStore {
	store := RoleStore{
		table:       table,
		changeTable: changeTable,
		roles:       make(map[int64]Role),
		userRoles:   make(map[int64]map[int64]struct{}),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *RoleStore) Get(id int64) (Role, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	role, ok := s.roles[id]
	return role, ok
}

func (s *RoleStore) Create(m *Role) error {
	change := roleChange{
		BaseChange: BaseChange{Type: CreateChange},
		Role:       *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Role
	return nil
}

func (s *RoleStore) Update(m *Role) error {
	change := roleChange{
		BaseChange: BaseChange{Type: UpdateChange},
		Role:       *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Role
	return nil
}

func (s *RoleStore) Delete(id int64) error {
	change := roleChange{
		BaseChange: BaseChange{Type: DeleteChange},
		Role:       Role{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *RoleStore) getLocker() sync.Locker {
	return &s.mutex
}

func (s *RoleStore) setupChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *RoleStore) loadChangeGapTx(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "code"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *RoleStore) scanChange(scan Scanner) (Change, error) {
	role := roleChange{}
	err := scan.Scan(
		&role.BaseChange.ID, &role.Type, &role.Time,
		&role.Role.ID, &role.Code,
	)
	return &role, err
}

func (s *RoleStore) saveChangeTx(tx *sql.Tx, change Change) error {
	role := change.(*roleChange)
	role.Time = time.Now().Unix()
	switch role.Type {
	case CreateChange:
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s" ("code") VALUES ($1)`,
				s.table,
			),
			role.Code,
		)
		if err != nil {
			return err
		}
		role.Role.ID, err = res.LastInsertId()
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.roles[role.Role.ID]; !ok {
			return fmt.Errorf(
				"role with id = %d does not exists",
				role.Role.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET "code" = $1 WHERE "id" = $2`,
				s.table,
			),
			role.Code, role.Role.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.roles[role.Role.ID]; !ok {
			return fmt.Errorf(
				"role with id = %d does not exists",
				role.Role.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1`,
				s.table,
			),
			role.Role.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			role.Type,
		)
	}
	res, err := tx.Exec(
		fmt.Sprintf(
			`INSERT INTO "%s"`+
				` ("change_type", "change_time", "id", "code")`+
				` VALUES ($1, $2, $3, $4)`,
			s.changeTable,
		),
		role.Type, role.Time, role.Role.ID, role.Code,
	)
	if err != nil {
		return err
	}
	role.BaseChange.ID, err = res.LastInsertId()
	return err
}

func (s *RoleStore) applyChange(change Change) {
	role := change.(*roleChange)
	switch role.Type {
	case UpdateChange:
		fallthrough
	case CreateChange:
		s.roles[role.Role.ID] = role.Role
	case DeleteChange:
		delete(s.roles, role.Role.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			role.Type,
		))
	}
}

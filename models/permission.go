package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Permission struct {
	ID       int64  `json:"" db:"id"`
	RoleCode string `json:"" db:"role_code"`
	Code     string `json:"" db:"code"`
}

type permissionChange struct {
	BaseChange
	Permission
}

type PermissionStore struct {
	Manager         *ChangeManager
	table           string
	changeTable     string
	permissions     map[int64]Permission
	rolePermissions map[int64]map[int64]struct{}
	mutex           sync.RWMutex
}

func NewPermissionStore(
	db *sql.DB, table, changeTable string,
) *PermissionStore {
	store := PermissionStore{
		table:           table,
		changeTable:     changeTable,
		permissions:     make(map[int64]Permission),
		rolePermissions: make(map[int64]map[int64]struct{}),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *PermissionStore) Get(id int64) (Permission, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	permission, ok := s.permissions[id]
	return permission, ok
}

func (s *PermissionStore) Create(m *Permission) error {
	change := permissionChange{
		BaseChange: BaseChange{Type: CreateChange},
		Permission: *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Permission
	return nil
}

func (s *PermissionStore) Update(m *Permission) error {
	change := permissionChange{
		BaseChange: BaseChange{Type: UpdateChange},
		Permission: *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Permission
	return nil
}

func (s *PermissionStore) Delete(id int64) error {
	change := permissionChange{
		BaseChange: BaseChange{Type: DeleteChange},
		Permission: Permission{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *PermissionStore) getLocker() sync.Locker {
	return &s.mutex
}

func (s *PermissionStore) setupChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *PermissionStore) loadChangeGapTx(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "role_code", "code"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *PermissionStore) scanChange(scan Scanner) (Change, error) {
	permission := permissionChange{}
	err := scan.Scan(
		&permission.BaseChange.ID, &permission.Type, &permission.Time,
		&permission.Permission.ID, &permission.RoleCode, &permission.Code,
	)
	return &permission, err
}

func (s *PermissionStore) saveChangeTx(tx *sql.Tx, change Change) error {
	permission := change.(*permissionChange)
	permission.Time = time.Now().Unix()
	switch permission.Type {
	case CreateChange:
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("role_code", "code")`+
					` VALUES ($1, $2)`,
				s.table,
			),
			permission.RoleCode, permission.Code,
		)
		if err != nil {
			return err
		}
		permission.Permission.ID, err = res.LastInsertId()
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.permissions[permission.Permission.ID]; !ok {
			return fmt.Errorf(
				"permission with id = %d does not exists",
				permission.Permission.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET`+
					` "role_code" = $1, "code" = $2`+
					` WHERE "id" = $3`,
				s.table,
			),
			permission.RoleCode, permission.Code,
			permission.Permission.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.permissions[permission.Permission.ID]; !ok {
			return fmt.Errorf(
				"permission with id = %d does not exists",
				permission.Permission.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1`,
				s.table,
			),
			permission.Permission.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			permission.Type,
		)
	}
	res, err := tx.Exec(
		fmt.Sprintf(
			`INSERT INTO "%s"`+
				` ("change_type", "change_time", "id", "role_code", "code")`+
				` VALUES ($1, $2, $3, $4, $5)`,
			s.changeTable,
		),
		permission.Type, permission.Time, permission.Permission.ID,
		permission.RoleCode, permission.Code,
	)
	if err != nil {
		return err
	}
	permission.BaseChange.ID, err = res.LastInsertId()
	return err
}

func (s *PermissionStore) applyChange(change Change) {
	permission := change.(*permissionChange)
	switch permission.Type {
	case UpdateChange:
		fallthrough
	case CreateChange:
		s.permissions[permission.Permission.ID] = permission.Permission
	case DeleteChange:
		delete(s.permissions, permission.Permission.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			permission.Type,
		))
	}
}

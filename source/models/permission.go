package models

import (
	"database/sql"
	"fmt"
	"time"
)

type Permission struct {
	ID   int64  `db:"id"   json:""`
	Code string `db:"code" json:""`
}

type PermissionChange struct {
	Permission
	ChangeBase
}

type PermissionStore struct {
	Manager     *ChangeManager
	db          *sql.DB
	table       string
	changeTable string
	permissions map[int64]Permission
}

func NewPermissionStore(
	db *sql.DB, table, changeTable string,
) *PermissionStore {
	store := PermissionStore{
		db: db, table: table, changeTable: changeTable,
		permissions: make(map[int64]Permission),
	}
	store.Manager = NewChangeManager(&store)
	return &store
}

func (s *PermissionStore) GetDB() *sql.DB {
	return s.db
}

func (s *PermissionStore) ChangeTableName() string {
	return s.changeTable
}

func (s *PermissionStore) Create(m *Permission) error {
	change := PermissionChange{
		ChangeBase: ChangeBase{Type: CreateChange},
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
	change := PermissionChange{
		ChangeBase: ChangeBase{Type: UpdateChange},
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
	change := PermissionChange{
		ChangeBase: ChangeBase{Type: DeleteChange},
		Permission: Permission{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *PermissionStore) scanChange(scan RowScan) (Change, error) {
	change := &PermissionChange{}
	err := scan.Scan(
		&change.ChangeBase.ID, &change.Type, &change.Time,
		&change.Permission.ID, &change.Code,
	)
	if err != nil {
		return nil, err
	}
	return change, nil
}

func (s *PermissionStore) saveChangeTx(tx *sql.Tx, change Change) error {
	permission := change.(*PermissionChange)
	permission.Time = time.Now().Unix()
	switch permission.Type {
	case CreateChange:
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s" ("code") VALUES ($1)`,
				s.table,
			),
			permission.Code,
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
				`UPDATE "%s" SET "code" = $1 WHERE "id" = $2`,
				s.table,
			),
			permission.Code, permission.Permission.ID,
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
			`INSERT INTO "%s" `+
				`("change_type", "change_time", "id", "code") `+
				`VALUES ($1, $2, $3, $4)`,
			s.ChangeTableName(),
		),
		permission.Type, permission.Time,
		permission.Permission.ID, permission.Code,
	)
	if err != nil {
		return err
	}
	permission.ChangeBase.ID, err = res.LastInsertId()
	return err
}

func (s *PermissionStore) applyChange(change Change) {
	permissionChange := change.(*PermissionChange)
	permission := permissionChange.Permission
	switch permissionChange.Type {
	case CreateChange:
		s.permissions[permission.ID] = permission
	case UpdateChange:
		s.permissions[permission.ID] = permission
	case DeleteChange:
		delete(s.permissions, permission.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			permissionChange.Type,
		))
	}
}

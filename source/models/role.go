package models

import (
	"database/sql"
	"fmt"
)

type Role struct {
	ID   int64  `db:"id"   json:""`
	Code string `db:"code" json:""`
}

type RoleChange struct {
	Role
	ID   int64      `db:"change_id" json:""`
	Type ChangeType `db:"change_type" json:""`
}

type RoleStore struct {
	db          *sql.DB
	table       string
	changeTable string
	roles       map[int64]Role
}

func (c *RoleChange) ChangeID() int64 {
	return c.ID
}

func (c *RoleChange) ChangeType() ChangeType {
	return c.Type
}

func (c *RoleChange) ChangeData() interface{} {
	return c.Role
}

func (s *RoleStore) NewRoleStore(
	db *sql.DB, table, changeTable string,
) *RoleStore {
	return &RoleStore{
		db:          db,
		table:       table,
		changeTable: changeTable,
	}
}

func (s *RoleStore) GetDB() *sql.DB {
	return s.db
}

func (s *RoleStore) TableName() string {
	return s.table
}

func (s *RoleStore) ChangeTableName() string {
	return s.changeTable
}

func (s *RoleStore) scanChange(scan RowScan) (Change, error) {
	change := &RoleChange{}
	if err := scan.Scan(change); err != nil {
		return nil, err
	}
	return change, nil
}

func (s *RoleStore) applyChange(change Change) error {
	role := change.ChangeData().(Role)
	switch change.ChangeType() {
	case CreateChange:
		s.roles[role.ID] = role
	case UpdateChange:
		s.roles[role.ID] = role
	case DeleteChange:
		delete(s.roles, role.ID)
	default:
		return fmt.Errorf(
			"unsupported change type = %d", change.ChangeType(),
		)
	}
	return nil
}

// func (s *RoleStore) Create(role *Role) error {
// 	// Disable Garbage Collector
// 	gcPercent := debug.SetGCPercent(-1)
// 	defer debug.SetGCPercent(gcPercent)
// 	// Start transaction
// 	tx, err := s.db.Begin()
// 	if err != nil {
// 		return err
// 	}
// 	_, err = tx.Exec(
// 		fmt.Sprintf(`LOCK TABLE "%s"`, s.table),
// 	)
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	if err := s.SyncTx(tx); err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	// Create record
// 	res, err := tx.Exec(
// 		fmt.Sprintf(`INSERT INTO "%s" ("code") VALUES ($1)`, s.table),
// 		role.Code,
// 	)
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	role.ID, err = res.LastInsertId()
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	// Create change record
// 	res, err = tx.Exec(
// 		fmt.Sprintf(
// 			`INSERT INTO "%s" ("change_type", "id", "code") VALUES ($1, $2, $3)`,
// 			s.changeTable,
// 		),
// 		CreateChange, role.ID, role.Code,
// 	)
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	change := Change{Type: CreateChange}
// 	change.ID, err = res.LastInsertId()
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	if err := tx.Commit(); err != nil {
// 		return err
// 	}
// 	// Apply change
// 	s.applyChange(RoleChange{change, *role})
// 	return nil
// }
//
// func (s *RoleStore) Update(role *Role) error {
// 	// Disable Garbage Collector to decrease table lock time
// 	gcPercent := debug.SetGCPercent(-1)
// 	defer debug.SetGCPercent(gcPercent)
// 	// Start transaction
// 	tx, err := s.db.Begin()
// 	if err != nil {
// 		return err
// 	}
// 	_, err = tx.Exec(
// 		fmt.Sprintf(`LOCK TABLE "%s"`, s.table),
// 	)
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	if err := s.SyncTx(tx); err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	// Update record
// 	_, err = tx.Exec(
// 		fmt.Sprintf(
// 			`UPDATE "%s" SET "code" = $1 WHERE "id" = $2`,
// 			s.table,
// 		),
// 		role.Code, role.ID,
// 	)
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	// Create change record
// 	res, err := tx.Exec(
// 		fmt.Sprintf(
// 			`INSERT INTO "%s" ("change_type", "id", "code") VALUES ($1, $2, $3)`,
// 			s.changeTable,
// 		),
// 		UpdateChange, role.ID, role.Code,
// 	)
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	change := Change{Type: UpdateChange}
// 	change.ID, err = res.LastInsertId()
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	if err := tx.Commit(); err != nil {
// 		return err
// 	}
// 	// Apply change
// 	s.applyChange(RoleChange{change, *role})
// 	return nil
// }
//
// func (s *RoleStore) Delete(id int64) error {
// 	// Disable Garbage Collector
// 	gcPercent := debug.SetGCPercent(-1)
// 	defer debug.SetGCPercent(gcPercent)
// 	// Start transaction
// 	tx, err := s.db.Begin()
// 	if err != nil {
// 		return err
// 	}
// 	_, err = tx.Exec(
// 		fmt.Sprintf(`LOCK TABLE "%s"`, s.table),
// 	)
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	if err := s.SyncTx(tx); err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	// Find role by ID
// 	role, err := s.Get(id)
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	// Create record
// 	_, err = tx.Exec(
// 		fmt.Sprintf(`DELETE FROM "%s" WHERE "id" = $1`, s.table),
// 		role.ID,
// 	)
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	// Create change record
// 	res, err := tx.Exec(
// 		fmt.Sprintf(
// 			`INSERT INTO "%s" ("change_type", "id", "code") VALUES ($1, $2, $3)`,
// 			s.changeTable,
// 		),
// 		DeleteChange, role.ID, role.Code,
// 	)
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	change := Change{Type: DeleteChange}
// 	change.ID, err = res.LastInsertId()
// 	if err != nil {
// 		if err := tx.Rollback(); err != nil {
// 			panic(err)
// 		}
// 		return err
// 	}
// 	if err := tx.Commit(); err != nil {
// 		return err
// 	}
// 	// Apply change
// 	s.applyChange(RoleChange{change, role})
// 	return nil
// }

func (s *RoleStore) Get(id int64) (Role, error) {
	role, ok := s.roles[id]
	if !ok {
		return role, sql.ErrNoRows
	}
	return role, nil
}

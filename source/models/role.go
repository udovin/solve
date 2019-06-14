package models

import (
	"database/sql"
	"fmt"
	"runtime/debug"
	"sync"
)

type RoleStore struct {
	db           *sql.DB
	table        string
	changeTable  string
	roles        map[int64]Role
	mutex        sync.Mutex
	lastChangeID int64
}

type Role struct {
	ID   int64  `db:"id"   json:""`
	Code string `db:"code" json:""`
}

type RoleChange struct {
	Change
	Role
}

func (s *RoleStore) applyChange(change RoleChange) {
	switch change.Type {
	case CreateChange:
		s.roles[change.Role.ID] = change.Role
	case UpdateChange:
		s.roles[change.Role.ID] = change.Role
	case DeleteChange:
		delete(s.roles, change.Role.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type '%d'",
			change.Change.Type,
		))
	}
	s.lastChangeID = change.Change.ID
}

func (s *RoleStore) Sync() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	rows, err := s.db.Query(
		fmt.Sprintf(
			`SELECT * FROM "%s" WHERE "change_id" > $1 ORDER BY "change_id"`,
			s.changeTable,
		),
		s.lastChangeID,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			panic(err)
		}
	}()
	for rows.Next() {
		var change RoleChange
		if err := rows.Scan(change); err != nil {
			return err
		}
		s.applyChange(change)
	}
	return nil
}

func (s *RoleStore) SyncTx(tx *sql.Tx) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	rows, err := tx.Query(
		fmt.Sprintf(
			`SELECT * FROM "%s" WHERE "change_id" > $1 ORDER BY "change_id"`,
			s.changeTable,
		),
		s.lastChangeID,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			panic(err)
		}
	}()
	for rows.Next() {
		var change RoleChange
		if err := rows.Scan(change); err != nil {
			return err
		}
		s.applyChange(change)
	}
	return nil
}

func (s *RoleStore) Create(role *Role) error {
	// Disable Garbage Collector
	gcPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(gcPercent)
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		fmt.Sprintf(`LOCK TABLE "%s"`, s.table),
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	if err := s.SyncTx(tx); err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	// Create record
	res, err := tx.Exec(
		fmt.Sprintf(`INSERT INTO "%s" ("code") VALUES ($1)`, s.table),
		role.Code,
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	role.ID, err = res.LastInsertId()
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	// Create change record
	res, err = tx.Exec(
		fmt.Sprintf(
			`INSERT INTO "%s" ("change_type", "id", "code") VALUES ($1, $2, $3)`,
			s.changeTable,
		),
		CreateChange, role.ID, role.Code,
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	change := Change{Type: CreateChange}
	change.ID, err = res.LastInsertId()
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// Apply change
	s.applyChange(RoleChange{change, *role})
	return nil
}

func (s *RoleStore) Update(role *Role) error {
	// Disable Garbage Collector to decrease table lock time
	gcPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(gcPercent)
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		fmt.Sprintf(`LOCK TABLE "%s"`, s.table),
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	if err := s.SyncTx(tx); err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	// Update record
	_, err = tx.Exec(
		fmt.Sprintf(
			`UPDATE "%s" SET "code" = $1 WHERE "id" = $2`,
			s.table,
		),
		role.Code, role.ID,
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	// Create change record
	res, err := tx.Exec(
		fmt.Sprintf(
			`INSERT INTO "%s" ("change_type", "id", "code") VALUES ($1, $2, $3)`,
			s.changeTable,
		),
		UpdateChange, role.ID, role.Code,
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	change := Change{Type: UpdateChange}
	change.ID, err = res.LastInsertId()
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// Apply change
	s.applyChange(RoleChange{change, *role})
	return nil
}

func (s *RoleStore) Delete(id int64) error {
	// Disable Garbage Collector
	gcPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(gcPercent)
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		fmt.Sprintf(`LOCK TABLE "%s"`, s.table),
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	if err := s.SyncTx(tx); err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	// Find role by ID
	role, err := s.Get(id)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	// Create record
	_, err = tx.Exec(
		fmt.Sprintf(`DELETE FROM "%s" WHERE "id" = $1`, s.table),
		role.ID,
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	// Create change record
	res, err := tx.Exec(
		fmt.Sprintf(
			`INSERT INTO "%s" ("change_type", "id", "code") VALUES ($1, $2, $3)`,
			s.changeTable,
		),
		DeleteChange, role.ID, role.Code,
	)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	change := Change{Type: DeleteChange}
	change.ID, err = res.LastInsertId()
	if err != nil {
		if err := tx.Rollback(); err != nil {
			panic(err)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// Apply change
	s.applyChange(RoleChange{change, role})
	return nil
}

func (s *RoleStore) Get(id int64) (Role, error) {
	role, ok := s.roles[id]
	if !ok {
		return role, sql.ErrNoRows
	}
	return role, nil
}

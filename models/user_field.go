package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// Additional information about user
// like E-mail, first name, last name and etc
type UserField struct {
	ID     int64  `json:"" db:"id"`
	UserID int64  `json:"" db:"user_id"`
	Type   string `json:"" db:"type"`
	Data   string `json:"" db:"data"`
}

type userFieldChange struct {
	BaseChange
	UserField
}

// Store that caches database records about user fields
type UserFieldStore struct {
	Manager     *ChangeManager
	table       string
	changeTable string
	fields      map[int64]UserField
	userFields  map[int64]map[int64]struct{}
	mutex       sync.RWMutex
}

// Create new instance of UserFieldStore
func NewUserFieldStore(
	db *sql.DB, table, changeTable string,
) *UserFieldStore {
	store := UserFieldStore{
		table:       table,
		changeTable: changeTable,
		fields:      make(map[int64]UserField),
		userFields:  make(map[int64]map[int64]struct{}),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

// Get user field by field's ID
func (s *UserFieldStore) Get(id int64) (UserField, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	field, ok := s.fields[id]
	return field, ok
}

// Create user field with specified data
func (s *UserFieldStore) Create(m *UserField) error {
	change := userFieldChange{
		BaseChange: BaseChange{Type: CreateChange},
		UserField:  *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.UserField
	return nil
}

// Modify user field
// Modification will be applied to field with ID = m.ID
func (s *UserFieldStore) Update(m *UserField) error {
	change := userFieldChange{
		BaseChange: BaseChange{Type: UpdateChange},
		UserField:  *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.UserField
	return nil
}

// Delete user field with specified ID
func (s *UserFieldStore) Delete(id int64) error {
	change := userFieldChange{
		BaseChange: BaseChange{Type: DeleteChange},
		UserField:  UserField{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *UserFieldStore) GetLocker() sync.Locker {
	return &s.mutex
}

func (s *UserFieldStore) InitChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *UserFieldStore) LoadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "user_id", "name", "data"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *UserFieldStore) ScanChange(scan Scanner) (Change, error) {
	field := userFieldChange{}
	err := scan.Scan(
		&field.BaseChange.ID, &field.BaseChange.Type, &field.Time,
		&field.UserField.ID, &field.UserID, &field.UserField.Type,
		&field.Data,
	)
	return &field, err
}

func (s *UserFieldStore) SaveChange(tx *sql.Tx, change Change) error {
	field := change.(*userFieldChange)
	field.Time = time.Now().Unix()
	switch field.BaseChange.Type {
	case CreateChange:
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("user_id", "name", "data")`+
					` VALUES ($1, $2, $3)`,
				s.table,
			),
			field.UserID, field.UserField.Type, field.Data,
		)
		if err != nil {
			return err
		}
		field.UserField.ID, err = res.LastInsertId()
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.fields[field.UserField.ID]; !ok {
			return fmt.Errorf(
				"user field with id = %d does not exists",
				field.UserField.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s" SET`+
					` "user_id" = $1, "name" = $2, "data" = $3`+
					` WHERE "id" = $4`,
				s.table,
			),
			field.UserID, field.UserField.Type, field.Data,
			field.UserField.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.fields[field.UserField.ID]; !ok {
			return fmt.Errorf(
				"user field with id = %d does not exists",
				field.UserField.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1`,
				s.table,
			),
			field.UserField.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			field.UserField.Type,
		)
	}
	res, err := tx.Exec(
		fmt.Sprintf(
			`INSERT INTO "%s"`+
				` ("change_type", "change_time",`+
				` "id", "user_id", "name", "data")`+
				` VALUES ($1, $2, $3, $4, $5, $6)`,
			s.changeTable,
		),
		field.BaseChange.Type, field.Time, field.UserField.ID,
		field.UserID, field.UserField.Type, field.Data,
	)
	if err != nil {
		return err
	}
	field.BaseChange.ID, err = res.LastInsertId()
	return err
}

func (s *UserFieldStore) ApplyChange(change Change) {
	field := change.(*userFieldChange)
	switch field.BaseChange.Type {
	case UpdateChange:
		if oldField, ok := s.fields[field.UserField.ID]; ok {
			if oldField.UserID != field.UserID {
				if userFields, ok := s.userFields[oldField.UserID]; ok {
					delete(userFields, oldField.ID)
					if len(userFields) == 0 {
						delete(s.userFields, field.UserID)
					}
				}
			}
		}
		fallthrough
	case CreateChange:
		if _, ok := s.userFields[field.UserID]; !ok {
			s.userFields[field.UserID] = make(map[int64]struct{})
		}
		s.userFields[field.UserID][field.UserField.ID] = struct{}{}
		s.fields[field.UserField.ID] = field.UserField
	case DeleteChange:
		if userFields, ok := s.userFields[field.UserID]; ok {
			delete(userFields, field.UserField.ID)
			if len(userFields) == 0 {
				delete(s.userFields, field.UserID)
			}
		}
		delete(s.fields, field.UserField.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			field.BaseChange.Type,
		))
	}
}

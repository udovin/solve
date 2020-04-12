package models

import (
	"database/sql"
	"fmt"

	"github.com/udovin/solve/db"
)

// UserField contains additional information about user
// like E-mail, first name, last name and etc.
type UserField struct {
	ID     int64         `db:"id" json:""`
	UserID int64         `db:"user_id" json:""`
	Type   UserFieldType `db:"type" json:""`
	Data   string        `db:"data" json:""`
}

// ObjectID returns ID of user field.
func (o UserField) ObjectID() int64 {
	return o.ID
}

func (o UserField) clone() UserField {
	return o
}

// UserFieldType represents type of UserField.
type UserFieldType int

func (t UserFieldType) String() string {
	switch t {
	case EmailField:
		return "Email"
	case FirstNameField:
		return "FirstName"
	case LastNameField:
		return "LastName"
	case MiddleNameField:
		return "MiddleName"
	default:
		return fmt.Sprintf("UserFieldType(%d)", t)
	}
}

const (
	// EmailField represents field type for email address.
	EmailField UserFieldType = 1
	// FirstNameField represents field type for first name.
	FirstNameField UserFieldType = 2
	// LastNameField represents field type for last name.
	LastNameField UserFieldType = 3
	// MiddleName represents field type for middle name.
	MiddleNameField UserFieldType = 4
)

// UserFieldEvent represents user field event.
type UserFieldEvent struct {
	baseEvent
	UserField
}

// Object returns user field.
func (e UserFieldEvent) Object() db.Object {
	return e.UserField
}

// WithObject returns copy of event with replaced user field data.
func (e UserFieldEvent) WithObject(o db.Object) ObjectEvent {
	e.UserField = o.(UserField)
	return e
}

// Manager that caches database records about user fields.
type UserFieldManager struct {
	baseManager
	fields map[int64]UserField
	byUser indexInt64
}

// Get returns user field by ID.
func (m *UserFieldManager) Get(id int64) (UserField, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if field, ok := m.fields[id]; ok {
		return field.clone(), nil
	}
	return UserField{}, sql.ErrNoRows
}

// FindByUser returns user field by user ID.
func (m *UserFieldManager) FindByUser(userID int64) ([]UserField, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var fields []UserField
	for id := range m.byUser[userID] {
		if field, ok := m.fields[id]; ok {
			fields = append(fields, field.clone())
		}
	}
	return fields, nil
}

// CreateTx creates user field and returns copy with valid ID.
func (m *UserFieldManager) CreateTx(
	tx *sql.Tx, field UserField,
) (UserField, error) {
	event, err := m.createObjectEvent(tx, UserFieldEvent{
		makeBaseEvent(CreateEvent),
		field,
	})
	if err != nil {
		return UserField{}, err
	}
	return event.Object().(UserField), nil
}

// UpdateTx updates user field with specified ID.
func (m *UserFieldManager) UpdateTx(tx *sql.Tx, field UserField) error {
	_, err := m.createObjectEvent(tx, UserFieldEvent{
		makeBaseEvent(UpdateEvent),
		field,
	})
	return err
}

// DeleteTx deletes user field with specified ID.
func (m *UserFieldManager) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := m.createObjectEvent(tx, UserFieldEvent{
		makeBaseEvent(DeleteEvent),
		UserField{ID: id},
	})
	return err
}

func (m *UserFieldManager) reset() {
	m.fields = map[int64]UserField{}
	m.byUser = indexInt64{}
}

func (m *UserFieldManager) onCreateObject(o db.Object) {
	field := o.(UserField)
	m.fields[field.ID] = field
	m.byUser.Create(field.UserID, field.ID)
}

func (m *UserFieldManager) onDeleteObject(o db.Object) {
	field := o.(UserField)
	m.byUser.Delete(field.UserID, field.ID)
	delete(m.fields, field.ID)
}

func (m *UserFieldManager) onUpdateObject(o db.Object) {
	field := o.(UserField)
	if old, ok := m.fields[field.ID]; ok {
		if old.UserID != field.UserID {
			m.byUser.Delete(old.UserID, old.ID)
		}
	}
	m.onCreateObject(o)
}

// NewUserFieldManager creates new instance of user field manager.
func NewUserFieldManager(
	table, eventTable string, dialect db.Dialect,
) *UserFieldManager {
	impl := &UserFieldManager{}
	impl.baseManager = makeBaseManager(
		UserField{}, table, UserFieldEvent{}, eventTable, impl, dialect,
	)
	return impl
}

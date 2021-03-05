package models

import (
	"database/sql"
	"fmt"

	"github.com/udovin/solve/db"
)

// UserField contains additional information about user
// like E-mail, first name, last name and etc.
type UserField struct {
	ID     int64         `db:"id"`
	UserID int64         `db:"user_id"`
	Kind   UserFieldKind `db:"kind"`
	Data   string        `db:"data"`
}

// ObjectID returns ID of user field.
func (o UserField) ObjectID() int64 {
	return o.ID
}

// Clone creates copy of user field.
func (o UserField) Clone() UserField {
	return o
}

// UserFieldKind represents type of UserField.
type UserFieldKind int

func (t UserFieldKind) String() string {
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
		return fmt.Sprintf("UserFieldKind(%d)", t)
	}
}

const (
	// EmailField represents field kind for email address.
	EmailField UserFieldKind = 1
	// FirstNameField represents field kind for first name.
	FirstNameField UserFieldKind = 2
	// LastNameField represents field kind for last name.
	LastNameField UserFieldKind = 3
	// MiddleNameField represents field kind for middle name.
	MiddleNameField UserFieldKind = 4
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

// UserFieldStore that caches database records about user fields.
type UserFieldStore struct {
	baseStore
	fields map[int64]UserField
	byUser indexInt64
}

// Get returns user field by ID.
func (s *UserFieldStore) Get(id int64) (UserField, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if field, ok := s.fields[id]; ok {
		return field.Clone(), nil
	}
	return UserField{}, sql.ErrNoRows
}

// FindByUser returns user field by user ID.
func (s *UserFieldStore) FindByUser(userID int64) ([]UserField, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var fields []UserField
	for id := range s.byUser[userID] {
		if field, ok := s.fields[id]; ok {
			fields = append(fields, field.Clone())
		}
	}
	return fields, nil
}

// CreateTx creates user field and returns copy with valid ID.
func (s *UserFieldStore) CreateTx(
	tx *sql.Tx, field UserField,
) (UserField, error) {
	event, err := s.createObjectEvent(tx, UserFieldEvent{
		makeBaseEvent(CreateEvent),
		field,
	})
	if err != nil {
		return UserField{}, err
	}
	return event.Object().(UserField), nil
}

// UpdateTx updates user field with specified ID.
func (s *UserFieldStore) UpdateTx(tx *sql.Tx, field UserField) error {
	_, err := s.createObjectEvent(tx, UserFieldEvent{
		makeBaseEvent(UpdateEvent),
		field,
	})
	return err
}

// DeleteTx deletes user field with specified ID.
func (s *UserFieldStore) DeleteTx(tx *sql.Tx, id int64) error {
	_, err := s.createObjectEvent(tx, UserFieldEvent{
		makeBaseEvent(DeleteEvent),
		UserField{ID: id},
	})
	return err
}

func (s *UserFieldStore) reset() {
	s.fields = map[int64]UserField{}
	s.byUser = indexInt64{}
}

func (s *UserFieldStore) onCreateObject(o db.Object) {
	field := o.(UserField)
	s.fields[field.ID] = field
	s.byUser.Create(field.UserID, field.ID)
}

func (s *UserFieldStore) onDeleteObject(o db.Object) {
	field := o.(UserField)
	s.byUser.Delete(field.UserID, field.ID)
	delete(s.fields, field.ID)
}

func (s *UserFieldStore) onUpdateObject(o db.Object) {
	field := o.(UserField)
	if old, ok := s.fields[field.ID]; ok {
		if old.UserID != field.UserID {
			s.byUser.Delete(old.UserID, old.ID)
		}
	}
	s.onCreateObject(o)
}

// NewUserFieldStore creates new instance of user field store.
func NewUserFieldStore(
	table, eventTable string, dialect db.Dialect,
) *UserFieldStore {
	impl := &UserFieldStore{}
	impl.baseStore = makeBaseStore(
		UserField{}, table, UserFieldEvent{}, eventTable, impl, dialect,
	)
	return impl
}

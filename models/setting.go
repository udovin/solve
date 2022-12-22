package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

// Setting represents setting.
type Setting struct {
	baseObject
	Key   string `db:"key"`
	Value string `db:"value"`
}

// Clone creates copy of setting.
func (o Setting) Clone() Setting {
	return o
}

// SettingEvent represents setting event.
type SettingEvent struct {
	baseEvent
	Setting
}

// Object returns event compiler.
func (e SettingEvent) Object() Setting {
	return e.Setting
}

// SetObject sets event setting.
func (e *SettingEvent) SetObject(o Setting) {
	e.Setting = o
}

// SettingStore represents store for settings.
type SettingStore struct {
	baseStore[Setting, SettingEvent, *Setting, *SettingEvent]
	byKey *index[string, Setting, *Setting]
}

// GetByKey returns setting by specified key.
func (s *SettingStore) GetByKey(key string) (Setting, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byKey.Get(key) {
		if object, ok := s.objects[id]; ok {
			return object.Clone(), nil
		}
	}
	return Setting{}, sql.ErrNoRows
}

var _ baseStoreImpl[Setting] = (*SettingStore)(nil)

// NewSettingStore creates a new instance of SettingStore.
func NewSettingStore(db *gosql.DB, table, eventTable string) *SettingStore {
	impl := &SettingStore{
		byKey: newIndex(func(o Setting) string { return o.Key }),
	}
	impl.baseStore = makeBaseStore[Setting, SettingEvent](
		db, table, eventTable, impl, impl.byKey,
	)
	return impl
}

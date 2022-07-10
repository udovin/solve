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
	settings map[int64]Setting
	byKey    map[string]int64
}

// Get returns setting by specified ID.
func (s *SettingStore) Get(id int64) (Setting, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if setting, ok := s.settings[id]; ok {
		return setting.Clone(), nil
	}
	return Setting{}, sql.ErrNoRows
}

// GetByKey returns setting by specified key.

func (s *SettingStore) GetByKey(key string) (Setting, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if id, ok := s.byKey[key]; ok {
		if setting, ok := s.settings[id]; ok {
			return setting.Clone(), nil
		}
	}
	return Setting{}, sql.ErrNoRows
}

func (s *SettingStore) All() ([]Setting, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	var settings []Setting
	for _, setting := range s.settings {
		settings = append(settings, setting)
	}
	return settings, nil
}

func (s *SettingStore) reset() {
	s.settings = map[int64]Setting{}
	s.byKey = map[string]int64{}
}

func (s *SettingStore) onCreateObject(setting Setting) {
	s.settings[setting.ID] = setting
	s.byKey[setting.Key] = setting.ID
}

func (s *SettingStore) onDeleteObject(id int64) {
	if setting, ok := s.settings[id]; ok {
		delete(s.byKey, setting.Key)
		delete(s.settings, setting.ID)
	}
}

var _ baseStoreImpl[Setting] = (*SettingStore)(nil)

// NewSettingStore creates a new instance of SettingStore.
func NewSettingStore(db *gosql.DB, table, eventTable string) *SettingStore {
	impl := &SettingStore{}
	impl.baseStore = makeBaseStore[Setting, SettingEvent](
		db, table, eventTable, impl,
	)
	return impl
}

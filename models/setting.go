package models

import (
	"database/sql"

	"github.com/udovin/gosql"
)

// Setting represents setting.
type Setting struct {
	ID    int64  `db:"id"`
	Key   string `db:"key"`
	Value string `db:"value"`
}

// ObjectID returns ID of setting.
func (o Setting) ObjectID() int64 {
	return o.ID
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
	baseStore[Setting, SettingEvent]
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

func (s *SettingStore) makeObject(id int64) Setting {
	return Setting{ID: id}
}

func (s *SettingStore) makeObjectEvent(typ EventType) SettingEvent {
	return SettingEvent{baseEvent: makeBaseEvent(typ)}
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

// NewSettingStore creates a new instance of SettingStore.
func NewSettingStore(db *gosql.DB, table, eventTable string) *SettingStore {
	impl := &SettingStore{}
	impl.baseStore = makeBaseStore[Setting, SettingEvent](
		db, table, eventTable, impl,
	)
	return impl
}

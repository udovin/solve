package models

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

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

type Option[T any] struct {
	Value T
	Empty bool
}

func (o Option[T]) OrElse(v T) T {
	if o.Empty {
		return v
	}
	return o.Value
}

func Value[T any](v T) Option[T] {
	return Option[T]{Value: v}
}

func Empty[T any]() Option[T] {
	return Option[T]{Empty: true}
}

// SettingStore represents store for settings.
type SettingStore struct {
	cachedStore[Setting, SettingEvent, *Setting, *SettingEvent]
	byKey *index[string, Setting, *Setting]
}

// GetByKey returns setting by specified key.
func (s *SettingStore) GetByKey(key string) (Setting, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for id := range s.byKey.Get(key) {
		if object, ok := s.objects.Get(id); ok {
			return object.Clone(), nil
		}
	}
	return Setting{}, sql.ErrNoRows
}

func (s *SettingStore) GetBool(key string) (Option[bool], error) {
	setting, err := s.GetByKey(key)
	if err != nil {
		if err == sql.ErrNoRows {
			return Empty[bool](), nil
		}
		return Empty[bool](), err
	}
	switch strings.ToLower(strings.TrimSpace(setting.Value)) {
	case "t", "1", "true":
		return Value(true), nil
	case "f", "0", "false":
		return Value(false), nil
	case "":
		return Empty[bool](), nil
	}
	return Empty[bool](), fmt.Errorf("invalid bool: %q", setting.Value)
}

func (s *SettingStore) GetInt64(key string) (Option[int64], error) {
	setting, err := s.GetByKey(key)
	if err != nil {
		if err == sql.ErrNoRows {
			return Empty[int64](), nil
		}
		return Empty[int64](), err
	}
	value := strings.TrimSpace(setting.Value)
	if value == "" {
		return Empty[int64](), nil
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return Empty[int64](), err
	}
	return Value(intValue), nil
}

// NewSettingStore creates a new instance of SettingStore.
func NewSettingStore(db *gosql.DB, table, eventTable string) *SettingStore {
	impl := &SettingStore{
		byKey: newIndex(func(o Setting) (string, bool) { return o.Key, true }),
	}
	impl.cachedStore = makeCachedStore[Setting, SettingEvent](
		db, table, eventTable, impl, impl.byKey,
	)
	return impl
}

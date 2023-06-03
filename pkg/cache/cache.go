package cache

import (
	"container/list"
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/udovin/algo/futures"
)

type Ref[T any] interface {
	futures.Future[T]
	Release()
}

type Manager[K comparable, V any] interface {
	Load(key K) Ref[V]
	Delete(key K) bool
	Cleanup() int
	Len() int
}

type Storage[K comparable, V any] interface {
	Get(key K) (V, error)
	Delete(key K, value V) error
}

func NewManager[K comparable, V any](storage Storage[K, V]) Manager[K, V] {
	return &manager[K, V]{
		values:  map[K]*value[K, V]{},
		deleted: list.New(),
		storage: storage,
	}
}

type manager[K comparable, V any] struct {
	mutex   sync.RWMutex
	values  map[K]*value[K, V]
	deleted *list.List
	storage Storage[K, V]
}

// Load attempts to load a value with the given key,
// or reuses a cached value.
//
// Note that the value may not be ready and you must
// wait using Get().
//
// If the value is not required, Release() should be called.
func (m *manager[K, V]) Load(key K) Ref[V] {
	if v, ok := m.getFast(key); ok {
		return v
	}
	return m.getSlow(key)
}

// Delete marks the value as deleted and the next Load()
// call on this key will return the new value.
//
// The value will be completely removed when it is not used
// by anything when calling the Cleanup() method.
func (m *manager[K, V]) Delete(key K) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, ok := m.values[key]
	if !ok {
		return false
	}
	delete(m.values, key)
	m.deleted.PushBack(v)
	return true
}

func (m *manager[K, V]) Cleanup() int {
	var free []*value[K, V]
	func() {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		it := m.deleted.Front()
		for it != nil {
			v := it.Value.(*value[K, V])
			jt := it.Next()
			if atomic.LoadInt64(&v.counter) == 0 {
				free = append(free, v)
				m.deleted.Remove(it)
			}
			it = jt
		}
	}()
	for _, v := range free {
		value, err := v.Get(context.Background())
		if err == nil {
			m.storage.Delete(v.key, value)
		}
	}
	return len(free)
}

func (m *manager[K, V]) Len() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.values)
}

func (m *manager[K, V]) getFast(key K) (Ref[V], bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if v, ok := m.values[key]; ok {
		v.access = time.Now()
		return m.acquire(v), true
	}
	return nil, false
}

func (m *manager[K, V]) getSlow(key K) Ref[V] {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if v, ok := m.values[key]; ok {
		v.access = time.Now()
		return m.acquire(v)
	}
	v := &value[K, V]{key: key, access: time.Now()}
	m.values[key] = v
	v.value = futures.Call(func() (V, error) {
		value, err := m.storage.Get(key)
		if err != nil {
			// Delete value on failure.
			m.mutex.Lock()
			defer m.mutex.Unlock()
			// Ensure that value is not deleted yet.
			if c, ok := m.values[key]; ok && c == v {
				delete(m.values, key)
			}
		}
		return value, err
	})
	return m.acquire(v)
}

func (m *manager[K, V]) acquire(v *value[K, V]) Ref[V] {
	atomic.AddInt64(&v.counter, 1)
	return &valueRef[K, V]{value: v}
}

type value[K, V any] struct {
	key     K
	value   futures.Future[V]
	counter int64
	access  time.Time
}

func (v *value[K, V]) Get(ctx context.Context) (V, error) {
	return v.value.Get(ctx)
}

func (v *value[K, V]) Done() <-chan struct{} {
	return v.value.Done()
}

type valueRef[K, V any] struct {
	*value[K, V]
	released atomic.Bool
}

func (v *valueRef[K, V]) Release() {
	if v.released.CompareAndSwap(false, true) {
		atomic.AddInt64(&v.value.counter, -1)
	}
}

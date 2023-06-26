package cache

import (
	"container/list"
	"context"
	"sync"
	"sync/atomic"

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
	Actual(key K, value V) bool
	Delete(key K, value V) error
}

func NewManager[K comparable, V any](storage Storage[K, V]) Manager[K, V] {
	return &manager[K, V]{
		values:  map[K]*value[V]{},
		deleted: list.New(),
		storage: storage,
	}
}

type manager[K comparable, V any] struct {
	mutex   sync.RWMutex
	values  map[K]*value[V]
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
		value, err := v.Get(canceledContext)
		if err != nil || m.storage.Actual(key, value) {
			return v
		}
		m.delete(key, v.value)
		v.Release()
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
	m.deleted.PushBack(keyValue[K, V]{key: key, value: v})
	return true
}

func (m *manager[K, V]) Cleanup() int {
	var free []keyValue[K, V]
	func() {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		it := m.deleted.Front()
		for it != nil {
			v := it.Value.(keyValue[K, V])
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

func (m *manager[K, V]) getFast(key K) (*valueRef[V], bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if v, ok := m.values[key]; ok {
		return m.acquire(v), true
	}
	return nil, false
}

func (m *manager[K, V]) getSlow(key K) *valueRef[V] {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if v, ok := m.values[key]; ok {
		return m.acquire(v)
	}
	v := &value[V]{}
	m.values[key] = v
	ref := m.acquire(v)
	v.value = futures.Call(func() (V, error) {
		value, err := m.storage.Get(key)
		if err != nil {
			m.delete(key, v)
		}
		return value, err
	})
	return ref
}

func (m *manager[K, V]) delete(key K, value *value[V]) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if c, ok := m.values[key]; ok && c == value {
		delete(m.values, key)
	}
}

func (m *manager[K, V]) acquire(v *value[V]) *valueRef[V] {
	atomic.AddInt64(&v.counter, 1)
	return &valueRef[V]{value: v}
}

type keyValue[K, V any] struct {
	*value[V]
	key K
}

type value[V any] struct {
	value   futures.Future[V]
	counter int64
}

func (v *value[V]) Get(ctx context.Context) (V, error) {
	return v.value.Get(ctx)
}

func (v *value[V]) Done() <-chan struct{} {
	return v.value.Done()
}

type valueRef[V any] struct {
	*value[V]
	released atomic.Bool
}

func (v *valueRef[V]) Release() {
	if v.released.CompareAndSwap(false, true) {
		atomic.AddInt64(&v.value.counter, -1)
	}
}

var canceledContext context.Context

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	canceledContext = ctx
}

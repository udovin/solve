package cache

import (
	"context"
	"sync"
	"sync/atomic"
)

type Resource[T any] interface {
	Get() T
	Release()
}

type Storage[K any, V any] interface {
	Load(ctx context.Context, key K) (Resource[V], error)
}

type Manager[K comparable, V any] interface {
	Storage[K, V]
	// Delete deletes resource with the given key from cache.
	Delete(key K) bool
	// Len returns current amount of keys in cache.
	Len() int
}

func NewManager[K comparable, V any](storage Storage[K, V]) Manager[K, V] {
	return &manager[K, V]{
		cache:   map[K]*resource[V]{},
		futures: map[K]*inflightResource[V]{},
		storage: storage,
	}
}

type manager[K comparable, V any] struct {
	mutex   sync.RWMutex
	cache   map[K]*resource[V]
	futures map[K]*inflightResource[V]
	storage Storage[K, V]
}

// Load loads resource with the given key,
// or reuses a cached resource.
//
// If the resource is not required, Release() should be called.
func (m *manager[K, V]) Load(ctx context.Context, key K) (Resource[V], error) {
	if i, r, ok := m.getFast(key); ok {
		return m.getResource(ctx, i, r)
	}
	i, r, ok := m.getSlow(key)
	if ok {
		return m.getResource(ctx, i, r)
	}
	panicking := true
	defer func() {
		m.finish(key, i)
		if panicking {
			r.Release()
		}
	}()
	v, err := m.storage.Load(ctx, key)
	panicking = false
	if err != nil {
		i.err = err
		r.Release()
		return nil, err
	}
	i.resource.resource = v
	if o := m.set(key, i.resource); o != nil {
		o.Release()
	}
	return r, nil
}

// Delete deletes resource with the given key from cache.
func (m *manager[K, V]) Delete(key K) bool {
	if r := m.delete(key); r != nil {
		r.Release()
		return true
	}
	return false
}

// Len returns amount of cached values.
func (m *manager[K, V]) Len() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.cache)
}

func (m *manager[K, V]) getResource(ctx context.Context, i *inflightResource[V], r *resourceRef[V]) (*resourceRef[V], error) {
	if i != nil {
		select {
		case <-i.done:
			if i.err != nil {
				r.Release()
				return nil, i.err
			}
		case <-ctx.Done():
			r.Release()
			return nil, ctx.Err()
		}
	}
	return r, nil
}

func (m *manager[K, V]) getFast(key K) (*inflightResource[V], *resourceRef[V], bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if r, ok := m.cache[key]; ok {
		return nil, r.acquire(), true
	}
	if i, ok := m.futures[key]; ok {
		return i, i.acquire(), true
	}
	return nil, nil, false
}

func (m *manager[K, V]) getSlow(key K) (*inflightResource[V], *resourceRef[V], bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if r, ok := m.cache[key]; ok {
		return nil, r.acquire(), true
	}
	if i, ok := m.futures[key]; ok {
		return i, i.acquire(), true
	}
	i := &inflightResource[V]{
		done:     make(chan struct{}),
		resource: &resource[V]{counter: 1},
	}
	m.futures[key] = i
	return i, &resourceRef[V]{resource: i.resource}, false
}

func (m *manager[K, V]) set(key K, r *resource[V]) *resource[V] {
	atomic.AddInt64(&r.counter, 1)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	o := m.cache[key]
	m.cache[key] = r
	return o
}

func (m *manager[K, V]) delete(key K) *resource[V] {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	r, ok := m.cache[key]
	if ok {
		delete(m.cache, key)
	}
	return r
}

func (m *manager[K, V]) finish(key K, i *inflightResource[V]) {
	close(i.done)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, ok := m.futures[key]
	if ok && v == i {
		delete(m.futures, key)
	}
}

type resource[T any] struct {
	resource Resource[T]
	counter  int64
}

func (r *resource[T]) acquire() *resourceRef[T] {
	if atomic.AddInt64(&r.counter, 1) > 1 {
		return &resourceRef[T]{resource: r}
	}
	return nil
}

func (r *resource[T]) Release() {
	if atomic.AddInt64(&r.counter, -1) == 0 && r.resource != nil {
		r.resource.Release()
		r.resource = nil
	}
}

type resourceRef[T any] struct {
	*resource[T]
}

func (r *resourceRef[T]) Get() T {
	return r.resource.resource.Get()
}

func (r *resourceRef[T]) Release() {
	if r.resource != nil {
		r.resource.Release()
		r.resource = nil
	}
}

type inflightResource[T any] struct {
	done chan struct{}
	err  error
	*resource[T]
}

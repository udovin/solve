package cache

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/udovin/algo/futures"
)

type Resource[T any] interface {
	Get() T
	Release()
}

type ResourceFuture[T any] interface {
	futures.Future[T]
	Release()
}

type Storage[K any, V any] interface {
	Load(ctx context.Context, key K) (Resource[V], error)
}

type Manager[K comparable, V any] interface {
	Load(ctx context.Context, key K) ResourceFuture[V]
	LoadSync(ctx context.Context, key K) (Resource[V], error)
	// Delete deletes resource with the given key from cache.
	Delete(key K) bool
	// Len returns current amount of keys in cache.
	Len() int
}

func NewManager[K comparable, V any](storage Storage[K, V]) Manager[K, V] {
	return &manager[K, V]{
		cache:   map[K]*resource[V]{},
		futures: map[K]*resourceFuture[V]{},
		storage: storage,
	}
}

type manager[K comparable, V any] struct {
	mutex   sync.RWMutex
	cache   map[K]*resource[V]
	futures map[K]*resourceFuture[V]
	storage Storage[K, V]
}

// Load loads resource with the given key,
// or reuses a cached resource.
//
// If the resource is not required, Release() should be called.
func (m *manager[K, V]) Load(ctx context.Context, key K) ResourceFuture[V] {
	if r, ok := m.loadFast(key); ok {
		return r
	}
	return m.loadSlow(ctx, key)
}

// LoadSync synchronously loads resource with the given key,
// or reuses a cached resource.
//
// If the resource is not required, Release() should be called.
func (m *manager[K, V]) LoadSync(ctx context.Context, key K) (Resource[V], error) {
	r := m.Load(ctx, key)
	switch v := r.(type) {
	case *resourceRef[V]:
		return &resourceSyncRef[V]{v.resource}, nil
	case *resourceFutureRef[V]:
		select {
		case <-v.future.done:
			if err := v.future.err; err != nil {
				v.Release()
				return nil, err
			}
			return &resourceSyncRef[V]{v.future.resource}, nil
		default:
		}
		select {
		case <-v.future.done:
			if err := v.future.err; err != nil {
				v.Release()
				return nil, err
			}
			return &resourceSyncRef[V]{v.future.resource}, nil
		case <-ctx.Done():
			v.Release()
			return nil, ctx.Err()
		}
	default:
		panic(fmt.Errorf("unexpected resource future type: %T", v))
	}
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

func (m *manager[K, V]) loadFast(key K) (ResourceFuture[V], bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if r, ok := m.cache[key]; ok {
		r.acquire()
		return &resourceRef[V]{r}, true
	}
	if f, ok := m.futures[key]; ok {
		f.resource.acquire()
		return &resourceFutureRef[V]{f}, true
	}
	return nil, false
}

func (m *manager[K, V]) loadSlow(ctx context.Context, key K) ResourceFuture[V] {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if r, ok := m.cache[key]; ok {
		r.acquire()
		return &resourceRef[V]{r}
	}
	if f, ok := m.futures[key]; ok {
		f.resource.acquire()
		return &resourceFutureRef[V]{f}
	}
	done := make(chan struct{})
	f := &resourceFuture[V]{
		done:     done,
		resource: &resource[V]{counter: 2},
	}
	m.futures[key] = f
	go func() {
		panicking := true
		defer func() {
			if r := recover(); panicking {
				f.resource.resource = nil
				f.err = futures.PanicError{
					Value: r,
					Stack: debug.Stack(),
				}
			}
			close(done)
		}()
		f.resource.resource, f.err = m.storage.Load(ctx, key)
		if r := m.updateCache(key, f); r != nil {
			r.Release()
		}
		panicking = false
	}()
	return &resourceFutureRef[V]{f}
}

func (m *manager[K, V]) updateCache(key K, f *resourceFuture[V]) *resource[V] {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if v, ok := m.futures[key]; ok && v == f {
		delete(m.futures, key)
	}
	if f.err != nil {
		return f.resource
	}
	r := m.cache[key]
	m.cache[key] = f.resource
	return r
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

type resource[T any] struct {
	resource Resource[T]
	counter  int64
}

func (r *resource[T]) acquire() {
	if atomic.AddInt64(&r.counter, 1) <= 1 {
		panic("cannot acquire released resource")
	}
}

func (r *resource[T]) Release() {
	if atomic.AddInt64(&r.counter, -1) == 0 && r.resource != nil {
		r.resource.Release()
		r.resource = nil
	}
}

type resourceFuture[T any] struct {
	done     chan struct{}
	err      error
	resource *resource[T]
}

type resourceRef[T any] struct {
	resource *resource[T]
}

func (r *resourceRef[T]) Get(ctx context.Context) (T, error) {
	return r.resource.resource.Get(), nil
}

func (r *resourceRef[T]) Done() <-chan struct{} {
	return chanDone
}

func (r *resourceRef[T]) Release() {
	if r.resource != nil {
		r.resource.Release()
		r.resource = nil
	}
}

type resourceFutureRef[T any] struct {
	future *resourceFuture[T]
}

func (r *resourceFutureRef[T]) Get(ctx context.Context) (T, error) {
	var empty T
	select {
	case <-r.future.done:
		if r.future.err != nil {
			return empty, r.future.err
		}
		return r.future.resource.resource.Get(), nil
	default:
	}
	select {
	case <-r.future.done:
		if r.future.err != nil {
			return empty, r.future.err
		}
		return r.future.resource.resource.Get(), nil
	case <-ctx.Done():
		return empty, ctx.Err()
	}
}

func (r *resourceFutureRef[T]) Done() <-chan struct{} {
	return r.future.done
}

func (r *resourceFutureRef[T]) Release() {
	if r.future != nil {
		r.future.resource.Release()
		r.future = nil
	}
}

type resourceSyncRef[T any] struct {
	resource *resource[T]
}

func (r *resourceSyncRef[T]) Get() T {
	return r.resource.resource.Get()
}

func (r *resourceSyncRef[T]) Release() {
	if r.resource != nil {
		r.resource.Release()
		r.resource = nil
	}
}

var chanDone = make(chan struct{})

func init() {
	close(chanDone)
}

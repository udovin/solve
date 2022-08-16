package managers

import (
	"context"
)

type Future[T any] interface {
	Get(ctx context.Context) (T, error)
}

func Async[T any](fn func() (T, error)) Future[T] {
	done := make(chan struct{})
	f := future[T]{done: done}
	go func() {
		defer close(done)
		f.value, f.err = fn()
	}()
	return &f
}

func After[T any, V any](future Future[T], fn func(Future[T]) (V, error)) Future[V] {
	wrapFn := func() (V, error) {
		return fn(future)
	}
	return Async(wrapFn)
}

type future[T any] struct {
	done  <-chan struct{}
	value T
	err   error
}

func (f *future[T]) Get(ctx context.Context) (T, error) {
	select {
	case <-f.done:
		return f.value, f.err
	default:
	}
	select {
	case <-f.done:
		return f.value, f.err
	case <-ctx.Done():
		return f.value, ctx.Err()
	}
}

package cache

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/udovin/algo/futures"
)

type testResult struct {
	Value string
	Err   error
}

type testStorageImpl struct {
	values chan testResult
}

type testResource struct {
	value    string
	released atomic.Bool
}

func (r *testResource) Get() string {
	return r.value
}

func (r *testResource) Release() {
	if !r.released.CompareAndSwap(false, true) {
		panic("already released")
	}
}

func (s *testStorageImpl) Load(ctx context.Context, key int) (Resource[string], error) {
	select {
	case result := <-s.values:
		if result.Err != nil {
			return nil, result.Err
		}
		return &testResource{value: result.Value}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestCache(t *testing.T) {
	storage := &testStorageImpl{
		values: make(chan testResult),
	}
	manager := NewManager[int, string](storage)
	func() {
		fref1 := futures.Call(func() (Resource[string], error) { return manager.Load(context.Background(), 42) })
		fref2 := futures.Call(func() (Resource[string], error) { return manager.Load(context.Background(), 42) })
		select {
		case <-time.After(time.Second):
			t.Fatal("Storage blocked")
		case storage.values <- testResult{"test", nil}:
		}
		ref1, err := fref1.Get(context.Background())
		if err != nil {
			t.Fatal("Error:", err)
		}
		defer ref1.Release()
		ref2, err := fref2.Get(context.Background())
		if err != nil {
			t.Fatal("Error:", err)
		}
		defer ref2.Release()
		expectEqual(t, ref1.Get(), "test")
		expectEqual(t, ref2.Get(), "test")
		expectEqual(t, manager.Len(), 1)
		expectEqual(t, manager.Delete(42), true)
		expectEqual(t, manager.Delete(42), false)
		expectEqual(t, manager.Len(), 0)
		ref1.Release()
		ref2.Release()
	}()
	func() {
		fref1 := futures.Call(func() (Resource[string], error) { return manager.Load(context.Background(), 42) })
		fref2 := futures.Call(func() (Resource[string], error) { return manager.Load(context.Background(), 42) })
		time.Sleep(time.Millisecond)
		select {
		case <-time.After(time.Second):
			t.Fatal("Storage blocked")
		case storage.values <- testResult{"", fmt.Errorf("test")}:
		}
		ref1, err := fref1.Get(context.Background())
		if err == nil {
			ref1.Release()
			t.Fatal("Expected error")
		}
		ref2, err := fref2.Get(context.Background())
		if err == nil {
			ref2.Release()
			t.Fatal("Expected error")
		}
		expectEqual(t, manager.Len(), 0)
		expectEqual(t, manager.Delete(42), false)
	}()
}

func expectEqual[T comparable](tb testing.TB, value, expected T) {
	if value != expected {
		tb.Fatalf("Expected %v, but got %v", expected, value)
	}
}

package cache

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
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
		ref1 := manager.Load(context.Background(), 42)
		defer ref1.Release()
		ref2 := manager.Load(context.Background(), 42)
		defer ref2.Release()
		select {
		case <-time.After(time.Second):
			t.Fatal("Storage blocked")
		case storage.values <- testResult{"test", nil}:
		}
		if value, err := ref1.Get(context.Background()); err != nil {
			t.Fatal("Error:", err)
		} else {
			expectEqual(t, value, "test")
		}
		if value, err := ref2.Get(context.Background()); err != nil {
			t.Fatal("Error:", err)
		} else {
			expectEqual(t, value, "test")
		}
		ref3, err := manager.LoadSync(context.Background(), 42)
		if err != nil {
			t.Fatal("Error:", err)
		}
		defer ref3.Release()
		expectEqual(t, ref3.Get(), "test")
		expectEqual(t, manager.Len(), 1)
		expectEqual(t, manager.Delete(42), true)
		expectEqual(t, manager.Delete(42), false)
		expectEqual(t, manager.Len(), 0)
		ref1.Release()
		ref2.Release()
		ref3.Release()
	}()
	func() {
		ref1 := manager.Load(context.Background(), 42)
		defer ref1.Release()
		ref2 := manager.Load(context.Background(), 42)
		defer ref2.Release()
		time.Sleep(time.Millisecond)
		select {
		case <-time.After(time.Second):
			t.Fatal("Storage blocked")
		case storage.values <- testResult{"", fmt.Errorf("test")}:
		}
		if _, err := ref1.Get(context.Background()); err == nil {
			t.Fatalf("Expected error")
		} else {
			expectEqual(t, err.Error(), "test")
		}
		if _, err := ref2.Get(context.Background()); err == nil {
			t.Fatalf("Expected error")
		} else {
			expectEqual(t, err.Error(), "test")
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

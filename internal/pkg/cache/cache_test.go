package cache

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

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

type testResult struct {
	Value string
	Err   error
}

type testStorageChanImpl struct {
	values chan testResult
}

func (s *testStorageChanImpl) Load(ctx context.Context, key int) (Resource[string], error) {
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
	storage := &testStorageChanImpl{
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

type testStorageConstImpl struct{}

const (
	testSuccessKey = 0
	testErrorKey   = 1
	testPanicKey   = 2
)

func (s *testStorageConstImpl) Load(ctx context.Context, key int) (Resource[string], error) {
	switch key {
	case testSuccessKey:
		return &testResource{value: "success"}, nil
	case testErrorKey:
		return nil, fmt.Errorf("test error")
	case testPanicKey:
		panic("test panic")
	}
	return &testResource{value: fmt.Sprint(key)}, nil
}

func TestSyncCache(t *testing.T) {
	storage := &testStorageConstImpl{}
	manager := NewManager[int, string](storage)
	func() {
		r, err := manager.LoadSync(context.Background(), testSuccessKey)
		if err != nil {
			t.Fatal("Error:", err)
		}
		defer r.Release()
		manager.Delete(testSuccessKey)
	}()
	func() {
		_, err := manager.LoadSync(context.Background(), testErrorKey)
		if err == nil {
			t.Fatalf("Expected error")
		}
		manager.Delete(testErrorKey)
	}()
	func() {
		_, err := manager.LoadSync(context.Background(), testPanicKey)
		if err == nil {
			t.Fatalf("Expected error")
		}
		manager.Delete(testPanicKey)
	}()
}

func expectEqual[T comparable](tb testing.TB, value, expected T) {
	if value != expected {
		tb.Fatalf("Expected %v, but got %v", expected, value)
	}
}

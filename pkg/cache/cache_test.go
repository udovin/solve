package cache

import (
	"context"
	"fmt"
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

func (s *testStorageImpl) Get(key int) (string, error) {
	result := <-s.values
	return result.Value, result.Err
}

func (s *testStorageImpl) Delete(key int, value string) error {
	return nil
}

func TestCache(t *testing.T) {
	storage := &testStorageImpl{
		values: make(chan testResult),
	}
	manager := NewManager[int, string](storage)
	func() {
		ref1 := manager.Load(42)
		defer ref1.Release()
		ref2 := manager.Load(42)
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
		expectEqual(t, manager.Len(), 1)
		expectEqual(t, manager.Delete(42), true)
		expectEqual(t, manager.Len(), 0)
		expectEqual(t, manager.Cleanup(), 0)
		ref1.Release()
		expectEqual(t, manager.Cleanup(), 0)
		// Check that double release is ignored.
		ref1.Release()
		expectEqual(t, manager.Cleanup(), 0)
		ref2.Release()
		expectEqual(t, manager.Cleanup(), 1)
	}()
	func() {
		ref1 := manager.Load(42)
		defer ref1.Release()
		ref2 := manager.Load(42)
		defer ref2.Release()
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
		expectEqual(t, manager.Cleanup(), 0)
		ref1.Release()
		expectEqual(t, manager.Cleanup(), 0)
		ref2.Release()
		expectEqual(t, manager.Cleanup(), 0)
	}()
}

func expectEqual[T comparable](tb testing.TB, value, expected T) {
	if value != expected {
		tb.Fatalf("Expected %v, but got %v", expected, value)
	}
}

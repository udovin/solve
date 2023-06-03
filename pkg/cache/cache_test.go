package cache

import (
	"context"
	"testing"
	"time"
)

type testStorageImpl struct {
	values chan string
}

func (s *testStorageImpl) Get(key int) (string, error) {
	return <-s.values, nil
}

func (s *testStorageImpl) Delete(key int, value string) error {
	return nil
}

func TestCache(t *testing.T) {
	storage := &testStorageImpl{
		values: make(chan string),
	}
	manager := NewManager[int, string](storage)
	func() {
		ref := manager.Load(42)
		defer ref.Release()
		select {
		case <-time.After(time.Second):
			t.Fatal("Storage blocked")
		case storage.values <- "test":
		}
		value, err := ref.Get(context.Background())
		if err != nil {
			t.Fatal("Error:", err)
		}
		if value != "test" {
			t.Fatalf("Expected %q but got %q", "test", value)
		}
		if l := manager.Len(); l != 1 {
			t.Fatalf("Expected %d but got %d", 1, l)
		}
		if ok := manager.Delete(42); !ok {
			t.Fatalf("Cannot delete key %d", 42)
		}
		if l := manager.Len(); l != 0 {
			t.Fatalf("Expected %d but got %d", 0, l)
		}
		if count := manager.Cleanup(); count != 0 {
			t.Fatalf("Expected %d but got %d", 0, count)
		}
	}()
	if count := manager.Cleanup(); count != 1 {
		t.Fatalf("Expected %d but got %d", 1, count)
	}
}

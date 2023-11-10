package models

import (
	"reflect"
	"sync"
	"testing"

	"github.com/udovin/algo/btree"
)

func lessInt(lhs, rhs int) bool {
	return lhs < rhs
}

func testSetupObjects() btree.Map[int64, testObject] {
	objects := btree.NewMap[int64, testObject](lessInt64)
	objects.Set(1, testObject{ID: 1, testObjectBase: testObjectBase{Int: 0}})
	objects.Set(2, testObject{ID: 2, testObjectBase: testObjectBase{Int: 5}})
	objects.Set(3, testObject{ID: 3, testObjectBase: testObjectBase{Int: 1}})
	objects.Set(4, testObject{ID: 4, testObjectBase: testObjectBase{Int: 1}})
	objects.Set(5, testObject{ID: 5, testObjectBase: testObjectBase{Int: 2}})
	objects.Set(6, testObject{ID: 6, testObjectBase: testObjectBase{Int: 3}})
	objects.Set(7, testObject{ID: 7, testObjectBase: testObjectBase{Int: 1}})
	return objects
}

func testSetupIndex(objects btree.Map[int64, testObject]) *btreeIndex[int, testObject, *testObject] {
	index := newBTreeIndex[int, testObject, *testObject](func(o testObject) (int, bool) {
		return o.Int, o.Int != 0
	}, lessInt)
	index.Reset()
	for it := objects.Iter(); it.Next(); {
		index.Register(it.Value())
	}
	return index
}

func TestBTreeIndexFind(t *testing.T) {
	objects := testSetupObjects()
	index := testSetupIndex(objects)
	{
		iter := index.Find(1)
		result := []int64{}
		for iter.Next() {
			result = append(result, iter.ID())
		}
		if expected := []int64{3, 4, 7}; !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected: %v, got: %v", expected, result)
		}
	}
	{
		iter := index.Find(4)
		result := []int64{}
		for iter.Next() {
			result = append(result, iter.ID())
		}
		if expected := []int64{}; !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected: %v, got: %v", expected, result)
		}
	}
	{
		iter := index.Find(10)
		result := []int64{}
		for iter.Next() {
			result = append(result, iter.ID())
		}
		if expected := []int64{}; !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected: %v, got: %v", expected, result)
		}
	}
}

func TestBTreeIndexReverseFind(t *testing.T) {
	objects := testSetupObjects()
	index := testSetupIndex(objects)
	{
		iter := index.ReverseFind(1)
		result := []int64{}
		for iter.Next() {
			result = append(result, iter.ID())
		}
		if expected := []int64{7, 4, 3}; !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected: %v, got: %v", expected, result)
		}
	}
	{
		iter := index.ReverseFind(4)
		result := []int64{}
		for iter.Next() {
			result = append(result, iter.ID())
		}
		if expected := []int64{}; !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected: %v, got: %v", expected, result)
		}
	}
	{
		iter := index.ReverseFind(10)
		result := []int64{}
		for iter.Next() {
			result = append(result, iter.ID())
		}
		if expected := []int64{}; !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected: %v, got: %v", expected, result)
		}
	}
}

func TestBTreeIndexFindMulti(t *testing.T) {
	objects := testSetupObjects()
	index := testSetupIndex(objects)
	mutex := sync.Mutex{}
	func() {
		mutex.Lock()
		rows := btreeIndexFind(index, objects.Iter(), &mutex, 1, 3, 2, 4)
		defer rows.Close()
		result := []int64{}
		for rows.Next() {
			row := rows.Row()
			result = append(result, row.ID)
		}
		if expected := []int64{3, 4, 5, 6, 7}; !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected: %v, got: %v", expected, result)
		}
	}()
	func() {
		mutex.Lock()
		rows := btreeIndexFind(index, objects.Iter(), &mutex, 4)
		defer rows.Close()
		result := []int64{}
		for rows.Next() {
			row := rows.Row()
			result = append(result, row.ID)
		}
		if expected := []int64{}; !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected: %v, got: %v", expected, result)
		}
	}()
	func() {
		mutex.Lock()
		rows := btreeIndexFind(index, objects.Iter(), &mutex, 10)
		defer rows.Close()
		result := []int64{}
		for rows.Next() {
			row := rows.Row()
			result = append(result, row.ID)
		}
		if expected := []int64{}; !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected: %v, got: %v", expected, result)
		}
	}()
}

func TestBTreeIndexReverseFindMulti(t *testing.T) {
	objects := testSetupObjects()
	index := testSetupIndex(objects)
	mutex := sync.Mutex{}
	func() {
		mutex.Lock()
		rows := btreeIndexReverseFind(index, objects.Iter(), &mutex, 1, 3, 2, 4)
		defer rows.Close()
		result := []int64{}
		for rows.Next() {
			row := rows.Row()
			result = append(result, row.ID)
		}
		if expected := []int64{7, 6, 5, 4, 3}; !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected: %v, got: %v", expected, result)
		}
	}()
	func() {
		mutex.Lock()
		rows := btreeIndexReverseFind(index, objects.Iter(), &mutex, 4)
		defer rows.Close()
		result := []int64{}
		for rows.Next() {
			row := rows.Row()
			result = append(result, row.ID)
		}
		if expected := []int64{}; !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected: %v, got: %v", expected, result)
		}
	}()
	func() {
		mutex.Lock()
		rows := btreeIndexReverseFind(index, objects.Iter(), &mutex, 10)
		defer rows.Close()
		result := []int64{}
		for rows.Next() {
			row := rows.Row()
			result = append(result, row.ID)
		}
		if expected := []int64{}; !reflect.DeepEqual(result, expected) {
			t.Fatalf("Expected: %v, got: %v", expected, result)
		}
	}()
}

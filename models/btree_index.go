package models

import (
	"container/heap"
	"math"
	"sync"

	"github.com/udovin/algo/btree"
	"github.com/udovin/solve/db"
)

type btreeIndexKey[T any] struct {
	Key T
	ID  int64
}

type btreeIndex[K any, T any, TPtr db.ObjectPtr[T]] struct {
	less func(K, K) bool
	key  func(T) (K, bool)
	m    btree.Map[btreeIndexKey[K], struct{}]
}

func newBTreeIndex[K comparable, T any, TPtr ObjectPtr[T]](
	key func(T) (K, bool), less func(K, K) bool,
) *btreeIndex[K, T, TPtr] {
	return &btreeIndex[K, T, TPtr]{key: key, less: less}
}

func (i *btreeIndex[K, T, TPtr]) Reset() {
	i.m = btree.NewMap[btreeIndexKey[K], struct{}](i.mapLess)
}

func (i *btreeIndex[K, T, TPtr]) Register(object T) {
	key, ok := i.key(object)
	if !ok {
		return
	}
	id := TPtr(&object).ObjectID()
	i.m.Set(btreeIndexKey[K]{Key: key, ID: id}, struct{}{})
}

func (i *btreeIndex[K, T, TPtr]) Deregister(object T) {
	key, ok := i.key(object)
	if !ok {
		return
	}
	id := TPtr(&object).ObjectID()
	i.m.Delete(btreeIndexKey[K]{Key: key, ID: id})
}

func (i *btreeIndex[K, T, TPtr]) Find(key K) *btreeIndexIter[K] {
	return &btreeIndexIter[K]{
		iter: i.m.Iter(),
		key:  key,
		less: i.less,
	}
}

func (i *btreeIndex[K, T, TPtr]) ReverseFind(key K) *btreeIndexReverseIter[K] {
	return &btreeIndexReverseIter[K]{
		iter: i.m.Iter(),
		key:  key,
		less: i.less,
	}
}

func (i *btreeIndex[K, T, TPtr]) mapLess(lhs, rhs btreeIndexKey[K]) bool {
	if i.less(lhs.Key, rhs.Key) {
		return true
	}
	if i.less(rhs.Key, lhs.Key) {
		return false
	}
	return lhs.ID < rhs.ID
}

type btreeIndexIter[K any] struct {
	iter   btree.MapIter[btreeIndexKey[K], struct{}]
	key    K
	less   func(K, K) bool
	seeked bool
}

func (i *btreeIndexIter[K]) Next() bool {
	if !i.seeked {
		i.seeked = true
		if !i.iter.Seek(btreeIndexKey[K]{Key: i.key, ID: math.MinInt64}) {
			return false
		}
	} else {
		if !i.iter.Next() {
			return false
		}
	}
	return !i.less(i.key, i.iter.Key().Key)
}

func (r *btreeIndexIter[K]) ID() int64 {
	return r.iter.Key().ID
}

type btreeIndexReverseIter[K any] struct {
	iter   btree.MapIter[btreeIndexKey[K], struct{}]
	key    K
	less   func(K, K) bool
	seeked bool
}

func (i *btreeIndexReverseIter[K]) Next() bool {
	if !i.seeked {
		i.seeked = true
		if !i.iter.Seek(btreeIndexKey[K]{Key: i.key, ID: math.MaxInt64}) {
			if !i.iter.Last() {
				return false
			}
		} else {
			if !i.iter.Prev() {
				return false
			}
		}
	} else {
		if !i.iter.Prev() {
			return false
		}
	}
	return !i.less(i.iter.Key().Key, i.key)
}

func (r *btreeIndexReverseIter[K]) ID() int64 {
	return r.iter.Key().ID
}

type btreeIndexRows[T any, TPtr ObjectPtr[T]] struct {
	iter  btree.MapIter[int64, T]
	iters heap.Interface
	mutex sync.Locker
}

func btreeIndexFind[K any, T any, TPtr ObjectPtr[T]](
	index *btreeIndex[K, T, TPtr],
	objects btree.MapIter[int64, T],
	locker sync.Locker,
	keys ...K,
) *btreeIndexRows[T, TPtr] {
	var iters indexIterHeap
	for _, key := range keys {
		it := index.Find(key)
		if it.Next() {
			iters = append(iters, it)
		}
	}
	heap.Init(&iters)
	return &btreeIndexRows[T, TPtr]{
		iter:  objects,
		iters: &iters,
		mutex: locker,
	}
}

func btreeIndexReverseFind[K any, T any, TPtr ObjectPtr[T]](
	index *btreeIndex[K, T, TPtr],
	objects btree.MapIter[int64, T],
	locker sync.Locker,
	keys ...K,
) *btreeIndexRows[T, TPtr] {
	var iters indexIterReverseHeap
	for _, key := range keys {
		it := index.ReverseFind(key)
		if it.Next() {
			iters = append(iters, it)
		}
	}
	heap.Init(&iters)
	return &btreeIndexRows[T, TPtr]{
		iter:  objects,
		iters: &iters,
		mutex: locker,
	}
}

func (r *btreeIndexRows[T, TPtr]) Next() bool {
	for r.iters.Len() > 0 {
		it := heap.Pop(r.iters).(indexIter)
		id := it.ID()
		if it.Next() {
			heap.Push(r.iters, it)
		}
		if r.iter.Seek(id) && r.iter.Key() == id {
			return true
		}
	}
	return false
}

func (r *btreeIndexRows[T, TPtr]) Row() T {
	value := r.iter.Value()
	return TPtr(&value).Clone()
}

func (r *btreeIndexRows[T, TPtr]) Err() error {
	return nil
}

func (r *btreeIndexRows[T, TPtr]) Close() error {
	if r.mutex == nil {
		return nil
	}
	r.mutex.Unlock()
	r.mutex = nil
	return nil
}

type indexIter interface {
	Next() bool
	ID() int64
}

type indexIterHeap []indexIter

func (a indexIterHeap) Len() int {
	return len(a)
}

func (a indexIterHeap) Less(i, j int) bool {
	return a[i].ID() < a[j].ID()
}

func (a indexIterHeap) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a *indexIterHeap) Push(x any) {
	*a = append(*a, x.(indexIter))
}

func (a *indexIterHeap) Pop() any {
	it := (*a)[len(*a)-1]
	*a = (*a)[:len(*a)-1]
	return it
}

type indexIterReverseHeap []indexIter

func (a indexIterReverseHeap) Len() int {
	return len(a)
}

func (a indexIterReverseHeap) Less(i, j int) bool {
	return a[j].ID() < a[i].ID()
}

func (a indexIterReverseHeap) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a *indexIterReverseHeap) Push(x any) {
	*a = append(*a, x.(indexIter))
}

func (a *indexIterReverseHeap) Pop() any {
	it := (*a)[len(*a)-1]
	*a = (*a)[:len(*a)-1]
	return it
}

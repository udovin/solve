package db

import (
	"container/list"
	"database/sql"
	"fmt"
	"math"
	"sync"
	"time"
)

// ChangeConsumer represents consumer for changes
type ChangeConsumer interface {
	Consume(tx *sql.Tx, fn func(Change) error) error
}

type changeGap struct {
	Begin, End int64
	Time       time.Time
}

type changeConsumer struct {
	store ChangeStore
	endID int64
	gaps  *list.List
	mutex sync.Mutex
}

func (c *changeConsumer) Consume(tx *sql.Tx, fn func(Change) error) error {
	c.skipOldGaps()
	if err := c.loadGapsChanges(tx, fn); err != nil {
		return err
	}
	return c.loadNewChanges(tx, fn)
}

// Some transactions may failure and such gaps will never been removed
// so we should skip this gaps after some other changes
const changeGapSkipWindow = 5000

// If there are no many changes we will do many useless requests to change
// store, so we should remove gaps by timeout
const changeGapSkipTimeout = 2 * time.Minute

func (c *changeConsumer) skipOldGaps() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	window := c.endID - changeGapSkipWindow
	timeout := time.Now().Add(-changeGapSkipTimeout)
	for c.gaps.Front() != nil {
		curr := c.gaps.Front().Value.(changeGap)
		if curr.End > window && curr.Time.After(timeout) {
			break
		}
		c.gaps.Remove(c.gaps.Front())
	}
}

func (c *changeConsumer) loadGapsChanges(
	tx *sql.Tx, fn func(Change) error,
) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for it := c.gaps.Front(); it != nil; {
		var err error
		if it, err = c.loadGapChanges(tx, it, fn); err != nil {
			return err
		}
	}
	return nil
}

func (c *changeConsumer) loadGapChanges(
	tx *sql.Tx, it *list.Element, fn func(Change) error,
) (*list.Element, error) {
	jt := it.Next()
	gap := it.Value.(changeGap)
	rows, err := c.store.LoadChanges(tx, gap.Begin, gap.End)
	if err != nil {
		return nil, err
	}
	prevID := gap.Begin - 1
	for rows.Next() {
		change := rows.Change()
		if change.ChangeID() <= prevID {
			_ = rows.Close()
			panic(fmt.Errorf(
				"change %v should have ID greater than %d",
				change, prevID,
			))
		}
		if change.ChangeID() >= gap.End {
			_ = rows.Close()
			panic(fmt.Errorf(
				"change %v should have ID less than %d",
				change, gap.End,
			))
		}
		if err := fn(change); err != nil {
			_ = rows.Close()
			return nil, err
		}
		prevID = change.ChangeID()
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return jt, rows.Err()
}

func (c *changeConsumer) loadNewChanges(
	tx *sql.Tx, fn func(Change) error,
) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	rows, err := c.store.LoadChanges(tx, c.endID, math.MaxInt64)
	if err != nil {
		return err
	}
	for rows.Next() {
		change := rows.Change()
		if change.ChangeID() < c.endID {
			_ = rows.Close()
			panic(fmt.Errorf(
				"change %v should have ID not less than %v",
				change, c.endID,
			))
		}
		if err := fn(change); err != nil {
			_ = rows.Close()
			return err
		}
		if c.endID < change.ChangeID() {
			c.gaps.PushBack(changeGap{
				Begin: c.endID,
				End:   change.ChangeID(),
				Time:  change.ChangeTime(),
			})
		}
		c.endID = change.ChangeID() + 1
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return rows.Err()
}

func NewChangeConsumer(store ChangeStore, beginID int64) ChangeConsumer {
	return &changeConsumer{
		store: store,
		endID: beginID,
		gaps:  list.New(),
	}
}

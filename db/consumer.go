package db

import (
	"container/list"
	"database/sql"
	"fmt"
	"math"
	"sync"
	"time"
)

// EventConsumer represents consumer for events
type EventConsumer interface {
	BeginID() int64
	Consume(tx *sql.Tx, fn func(Event) error) error
}

type eventGap struct {
	Begin, End int64
	Time       time.Time
}

type eventConsumer struct {
	store EventROStore
	endID int64
	gaps  *list.List
	mutex sync.Mutex
}

func (c *eventConsumer) BeginID() int64 {
	if it := c.gaps.Front(); it != nil {
		return it.Value.(eventGap).Begin
	}
	return c.endID
}

func (c *eventConsumer) Consume(tx *sql.Tx, fn func(Event) error) error {
	c.skipOldGaps()
	if err := c.loadGapsChanges(tx, fn); err != nil {
		return err
	}
	return c.loadNewChanges(tx, fn)
}

// Some transactions may failure and such gaps will never been removed
// so we should skip this gaps after some other changes
const eventGapSkipWindow = 5000

// If there are no many changes we will do many useless requests to change
// store, so we should remove gaps by timeout
const eventGapSkipTimeout = 2 * time.Minute

func (c *eventConsumer) skipOldGaps() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	window := c.endID - eventGapSkipWindow
	timeout := time.Now().Add(-eventGapSkipTimeout)
	for c.gaps.Front() != nil {
		curr := c.gaps.Front().Value.(eventGap)
		if curr.End > window && curr.Time.After(timeout) {
			break
		}
		c.gaps.Remove(c.gaps.Front())
	}
}

func (c *eventConsumer) loadGapsChanges(
	tx *sql.Tx, fn func(Event) error,
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

func (c *eventConsumer) loadGapChanges(
	tx *sql.Tx, it *list.Element, fn func(Event) error,
) (*list.Element, error) {
	jt := it.Next()
	gap := it.Value.(eventGap)
	rows, err := c.store.LoadEvents(tx, gap.Begin, gap.End)
	if err != nil {
		return nil, err
	}
	prevID := gap.Begin - 1
	for rows.Next() {
		event := rows.Event()
		if event.EventID() <= prevID {
			_ = rows.Close()
			panic(fmt.Errorf(
				"event %d should have ID greater than %d",
				event.EventID(), prevID,
			))
		}
		if event.EventID() >= gap.End {
			_ = rows.Close()
			panic(fmt.Errorf(
				"event %d should have ID less than %d",
				event.EventID(), gap.End,
			))
		}
		if err := fn(event); err != nil {
			_ = rows.Close()
			return nil, err
		}
		prevID = event.EventID()
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return jt, rows.Err()
}

func (c *eventConsumer) loadNewChanges(
	tx *sql.Tx, fn func(Event) error,
) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	rows, err := c.store.LoadEvents(tx, c.endID, math.MaxInt64)
	if err != nil {
		return err
	}
	for rows.Next() {
		event := rows.Event()
		if event.EventID() < c.endID {
			_ = rows.Close()
			panic(fmt.Errorf(
				"event %d should have ID not less than %d",
				event.EventID(), c.endID,
			))
		}
		if err := fn(event); err != nil {
			_ = rows.Close()
			return err
		}
		if c.endID < event.EventID() {
			c.gaps.PushBack(eventGap{
				Begin: c.endID,
				End:   event.EventID(),
				Time:  event.EventTime(),
			})
		}
		c.endID = event.EventID() + 1
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return rows.Err()
}

func NewEventConsumer(store EventROStore, beginID int64) EventConsumer {
	return &eventConsumer{store: store, endID: beginID, gaps: list.New()}
}

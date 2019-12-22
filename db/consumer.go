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
	// BeginEventID should return smallest ID of next possibly consumed event
	BeginEventID() int64
	// ConsumeEvents should consume new events
	ConsumeEvents(tx *sql.Tx, fn func(Event) error) error
}

// eventGap represents a gap in event sequence
type eventGap struct {
	beginID int64
	endID   int64
	time    time.Time
}

// eventConsumer represents a base implementation for EventConsumer
type eventConsumer struct {
	store EventROStore
	endID int64
	gaps  *list.List
	mutex sync.Mutex
}

// BeginEventID returns id of beginning event
func (c *eventConsumer) BeginEventID() int64 {
	if it := c.gaps.Front(); it != nil {
		return it.Value.(eventGap).beginID
	}
	return c.endID
}

// ConsumeEvents consumes new events from event store
func (c *eventConsumer) ConsumeEvents(tx *sql.Tx, fn func(Event) error) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if err := c.loadGapsChanges(tx, fn); err != nil {
		return err
	}
	return c.loadNewChanges(tx, fn)
}

// Some transactions may failure and such gaps will never been removed
// so we should skip this gaps after some other events
const eventGapSkipWindow = 5000

// If there are no many events we will do many useless requests to event
// store, so we should remove gaps by timeout
const eventGapSkipTimeout = 2 * time.Minute

func (c *eventConsumer) loadGapsChanges(
	tx *sql.Tx, fn func(Event) error,
) error {
	window := c.endID - eventGapSkipWindow
	timeout := time.Now().Add(-eventGapSkipTimeout)
	for it := c.gaps.Front(); it != nil; {
		jt := it.Next()
		if err := c.loadGapChanges(tx, it, fn, window, timeout); err != nil {
			return err
		}
		it = jt
	}
	return nil
}

func (c *eventConsumer) loadGapChanges(
	tx *sql.Tx, it *list.Element, fn func(Event) error,
	window int64, timeout time.Time,
) error {
	gap := it.Value.(eventGap)
	if gap.endID < window || gap.time.Before(timeout) {
		c.gaps.Remove(it)
		return nil
	}
	rows, err := c.store.LoadEvents(tx, gap.beginID, gap.endID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	prevID := gap.beginID - 1
	for rows.Next() {
		event := rows.Event()
		if event.EventID() <= prevID {
			_ = rows.Close()
			panic(fmt.Errorf(
				"event %d should have ID greater than %d",
				event.EventID(), prevID,
			))
		}
		if event.EventID() >= gap.endID {
			_ = rows.Close()
			panic(fmt.Errorf(
				"event %d should have ID less than %d",
				event.EventID(), gap.endID,
			))
		}
		if err := fn(event); err != nil {
			_ = rows.Close()
			return err
		}
		prevID = event.EventID()
		if event.EventID() > gap.beginID {
			newGap := eventGap{
				beginID: event.EventID() + 1,
				endID:   gap.endID,
				time:    gap.time,
			}
			gap.endID = event.EventID()
			it.Value = gap
			if newGap.beginID == newGap.endID {
				break
			}
			it = c.gaps.InsertAfter(newGap, it)
			gap = newGap
		} else {
			gap.beginID++
			if gap.beginID == gap.endID {
				c.gaps.Remove(it)
				break
			}
			it.Value = gap
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return rows.Err()
}

func (c *eventConsumer) loadNewChanges(
	tx *sql.Tx, fn func(Event) error,
) error {
	rows, err := c.store.LoadEvents(tx, c.endID, math.MaxInt64)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
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
				beginID: c.endID,
				endID:   event.EventID(),
				time:    event.EventTime(),
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
	return &eventConsumer{
		store: store,
		endID: beginID,
		gaps:  list.New(),
	}
}

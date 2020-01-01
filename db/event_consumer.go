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
	// BeginEventId should return smallest id of next possibly consumed event
	BeginEventId() int64
	// ConsumeEvents should consume new events
	ConsumeEvents(tx *sql.Tx, fn func(Event) error) error
}

// eventGap represents a gap in event sequence
type eventGap struct {
	beginId int64
	endId   int64
	time    time.Time
}

// eventConsumer represents a base implementation for EventConsumer
type eventConsumer struct {
	store EventROStore
	endId int64
	gaps  *list.List
	mutex sync.Mutex
}

// BeginEventId returns id of beginning event
func (c *eventConsumer) BeginEventId() int64 {
	if it := c.gaps.Front(); it != nil {
		return it.Value.(eventGap).beginId
	}
	return c.endId
}

// ConsumeEvents consumes new events from event store
func (c *eventConsumer) ConsumeEvents(tx *sql.Tx, fn func(Event) error) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if err := c.loadGapsChanges(tx, fn); err != nil {
		return err
	}
	if err := c.loadNewChanges(tx, fn); err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	}
	return nil
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
	window := c.endId - eventGapSkipWindow
	timeout := time.Now().Add(-eventGapSkipTimeout)
	for it := c.gaps.Front(); it != nil; {
		jt := it.Next()
		if err := c.loadGapChanges(tx, it, fn, window, timeout); err != nil {
			if err != sql.ErrNoRows {
				return err
			}
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
	if gap.endId < window || gap.time.Before(timeout) {
		c.gaps.Remove(it)
		return nil
	}
	rows, err := c.store.LoadEvents(tx, gap.beginId, gap.endId)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	prevId := gap.beginId - 1
	for rows.Next() {
		event := rows.Event()
		if event.EventId() <= prevId {
			return fmt.Errorf(
				"event %d should have id greater than %d",
				event.EventId(), prevId,
			)
		}
		if event.EventId() >= gap.endId {
			return fmt.Errorf(
				"event %d should have id less than %d",
				event.EventId(), gap.endId,
			)
		}
		if err := fn(event); err != nil {
			return err
		}
		prevId = event.EventId()
		if event.EventId() > gap.beginId {
			newGap := eventGap{
				beginId: event.EventId() + 1,
				endId:   gap.endId,
				time:    gap.time,
			}
			gap.endId = event.EventId()
			it.Value = gap
			if newGap.beginId == newGap.endId {
				break
			}
			it = c.gaps.InsertAfter(newGap, it)
			gap = newGap
		} else {
			gap.beginId++
			if gap.beginId == gap.endId {
				c.gaps.Remove(it)
				break
			}
			it.Value = gap
		}
	}
	return rows.Err()
}

func (c *eventConsumer) loadNewChanges(
	tx *sql.Tx, fn func(Event) error,
) error {
	rows, err := c.store.LoadEvents(tx, c.endId, math.MaxInt64)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		event := rows.Event()
		if event.EventId() < c.endId {
			return fmt.Errorf(
				"event %d should have id not less than %d",
				event.EventId(), c.endId,
			)
		}
		if err := fn(event); err != nil {
			return err
		}
		if c.endId < event.EventId() {
			c.gaps.PushBack(eventGap{
				beginId: c.endId,
				endId:   event.EventId(),
				time:    event.EventTime(),
			})
		}
		c.endId = event.EventId() + 1
	}
	return rows.Err()
}

// NewEventConsumer creates consumer for event store
func NewEventConsumer(store EventROStore, beginId int64) EventConsumer {
	return &eventConsumer{
		store: store,
		endId: beginId,
		gaps:  list.New(),
	}
}

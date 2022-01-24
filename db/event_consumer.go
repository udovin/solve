package db

import (
	"fmt"
	"sync"
	"time"

	"github.com/udovin/gosql"
)

// EventConsumer represents consumer for events.
type EventConsumer interface {
	// BeginEventID should return smallest ID of next possibly consumed event.
	BeginEventID() int64
	// ConsumeEvents should consume new events.
	ConsumeEvents(tx gosql.WeakTx, fn func(Event) error) error
}

// eventConsumer represents a base implementation for EventConsumer.
type eventConsumer struct {
	store  EventROStore
	ranges []EventRange
	mutex  sync.Mutex
}

// BeginEventID returns ID of beginning event.
func (c *eventConsumer) BeginEventID() int64 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.ranges[0].Begin
}

func (c *eventConsumer) removeEmptyRanges() {
	newLen := 0
	for i, rng := range c.ranges {
		if rng.Begin != rng.End {
			c.ranges[newLen] = c.ranges[i]
			newLen++
		}
	}
	c.ranges = c.ranges[:newLen]
	if len(c.ranges) > eventGapSkipWindow {
		c.ranges = c.ranges[len(c.ranges)-eventGapSkipWindow:]
	}
}

// ConsumeEvents consumes new events from event store.
func (c *eventConsumer) ConsumeEvents(tx gosql.WeakTx, fn func(Event) error) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	events, err := c.store.LoadEvents(tx, c.ranges)
	if err != nil {
		return err
	}
	defer func() {
		_ = events.Close()
	}()
	it := 0
	for events.Next() {
		event := events.Event()
		for it < len(c.ranges) && !c.ranges[it].contains(event.EventID()) {
			it++
		}
		if it == len(c.ranges) {
			return fmt.Errorf("invalid event ID: case 1")
		}
		if err := fn(event); err != nil {
			return err
		}
		if event.EventID() == c.ranges[it].Begin {
			c.ranges[it].Begin++
		} else {
			c.ranges = append(c.ranges, c.ranges[len(c.ranges)-1])
			for i := len(c.ranges) - 3; i >= it; i-- {
				c.ranges[i+1] = c.ranges[i]
			}
			c.ranges[it].End = event.EventID()
			c.ranges[it+1].Begin = event.EventID() + 1
		}
	}
	c.removeEmptyRanges()
	return events.Err()
}

// Some transactions may failure and such gaps will never been removed
// so we should skip this gaps after some other events.
const eventGapSkipWindow = 5000

// If there are no many events we will do many useless requests to event
// store, so we should remove gaps by timeout.
const eventGapSkipTimeout = 5 * time.Minute

// NewEventConsumer creates consumer for event store.
//
// TODO(udovin): Add support for gapSkipTimeout.
// TODO(udovin): Add support for limit.
func NewEventConsumer(store EventROStore, beginID int64) EventConsumer {
	return &eventConsumer{
		store:  store,
		ranges: []EventRange{{Begin: beginID}},
	}
}

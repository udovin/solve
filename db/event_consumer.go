package db

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EventConsumer represents consumer for events.
type EventConsumer[T any, TPtr EventPtr[T]] interface {
	// BeginEventID should return smallest ID of next possibly consumed event.
	BeginEventID() int64
	// ConsumeEvents should consume new events.
	ConsumeEvents(ctx context.Context, fn func(T) error) error
}

// eventConsumer represents a base implementation for EventConsumer.
type eventConsumer[T any, TPtr EventPtr[T]] struct {
	store  EventROStore[T]
	ranges []EventRange
	mutex  sync.Mutex
}

// BeginEventID returns ID of beginning event.
func (c *eventConsumer[T, TPtr]) BeginEventID() int64 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.ranges[0].Begin
}

func (c *eventConsumer[T, TPtr]) removeEmptyRanges() {
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
func (c *eventConsumer[T, TPtr]) ConsumeEvents(ctx context.Context, fn func(T) error) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	events, err := c.store.LoadEvents(ctx, c.ranges)
	if err != nil {
		return err
	}
	defer func() {
		_ = events.Close()
	}()
	it := 0
	for events.Next() {
		event := events.Row()
		eventID := TPtr(&event).EventID()
		for it < len(c.ranges) && !c.ranges[it].contains(eventID) {
			it++
		}
		if it == len(c.ranges) {
			return fmt.Errorf("invalid event ID: case 1")
		}
		if err := fn(event); err != nil {
			return err
		}
		if eventID == c.ranges[it].Begin {
			c.ranges[it].Begin++
		} else {
			c.ranges = append(c.ranges, c.ranges[len(c.ranges)-1])
			for i := len(c.ranges) - 3; i >= it; i-- {
				c.ranges[i+1] = c.ranges[i]
			}
			c.ranges[it].End = eventID
			c.ranges[it+1].Begin = eventID + 1
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
func NewEventConsumer[T any, TPtr EventPtr[T]](
	store EventROStore[T], beginID int64,
) EventConsumer[T, TPtr] {
	return &eventConsumer[T, TPtr]{
		store:  store,
		ranges: []EventRange{{Begin: beginID}},
	}
}

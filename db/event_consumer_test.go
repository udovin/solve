package db

import (
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/udovin/gosql"
)

type mockEvent struct {
	ID   int64 `db:"id"`
	Time int64 `db:"time"`
}

func (e mockEvent) String() string {
	return fmt.Sprintf("%d", e.ID)
}

func (e mockEvent) EventID() int64 {
	return e.ID
}

func (e mockEvent) EventTime() time.Time {
	if e.Time == 0 {
		return time.Now()
	}
	return time.Unix(e.Time, 0)
}

type mockEventStore struct {
	events []Event
}

func (s *mockEventStore) LastEventID(tx gosql.WeakTx) (int64, error) {
	return 0, nil
}

type eventSorter []Event

func (e eventSorter) Len() int {
	return len(e)
}

func (e eventSorter) Less(i, j int) bool {
	return e[i].EventID() < e[j].EventID()
}

func (e eventSorter) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (s *mockEventStore) LoadEvents(
	tx gosql.WeakTx, ranges []EventRange,
) (EventReader, error) {
	var events []Event
	for _, rng := range ranges {
		for _, event := range s.events {
			if event.EventID() >= rng.Begin && event.EventID() < rng.End {
				events = append(events, event)
			}
		}
	}
	if len(events) == 0 {
		return nil, sql.ErrNoRows
	}
	sort.Sort(eventSorter(events))
	return &mockEventReader{events: events}, nil
}

func (s *mockEventStore) FindEvents(
	tx *sql.Tx, where string, args ...any,
) (EventReader, error) {
	return nil, sql.ErrNoRows
}

type mockEventReader struct {
	events []Event
	event  Event
	pos    int
}

func (r *mockEventReader) Next() bool {
	if r.pos < len(r.events) {
		r.event = r.events[r.pos]
		r.pos++
		return true
	}
	return false
}

func (r *mockEventReader) Event() Event {
	return r.event
}

func (r *mockEventReader) Close() error {
	return nil
}

func (r *mockEventReader) Err() error {
	return nil
}

func TestEventConsumer(t *testing.T) {
	groups := [][]Event{
		{
			mockEvent{ID: 1}, mockEvent{ID: 2}, mockEvent{ID: 3},
		},
		{
			mockEvent{ID: 5}, mockEvent{ID: 6}, mockEvent{ID: 8},
		},
		{
			mockEvent{ID: 4}, mockEvent{ID: 7}, mockEvent{ID: 100},
		},
		{
			mockEvent{ID: 50}, mockEvent{ID: 75}, mockEvent{ID: 101},
		},
		{
			mockEvent{ID: 51}, mockEvent{ID: 74}, mockEvent{ID: 102},
		},
		{
			mockEvent{ID: 25}, mockEvent{ID: 97}, mockEvent{ID: 98},
			mockEvent{ID: 99}, mockEvent{ID: 103},
		},
		{
			mockEvent{ID: 27}, mockEvent{ID: 28}, mockEvent{ID: 29},
			mockEvent{ID: 104},
		},
		{
			mockEvent{ID: 26},
		},
	}
	store := &mockEventStore{}
	consumer := NewEventConsumer(store, 1)
	var result, answer []mockEvent
	usedIDs := map[int64]struct{}{}
	currID := int64(1)
	for _, group := range groups {
		for _, event := range group {
			store.events = append(store.events, event)
			answer = append(answer, event.(mockEvent))
		}
		errConsume := fmt.Errorf("consuming error")
		if err := consumer.ConsumeEvents(nil, func(event Event) error {
			return errConsume
		}); err != errConsume {
			t.Fatal(err)
		}
		if err := consumer.ConsumeEvents(nil, func(event Event) error {
			result = append(result, event.(mockEvent))
			usedIDs[event.EventID()] = struct{}{}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		for {
			if _, ok := usedIDs[currID]; !ok {
				break
			}
			currID++
		}
		if consumer.BeginEventID() != currID {
			t.Fatalf("Expected %d, got %d", currID, consumer.BeginEventID())
		}
	}
	if !reflect.DeepEqual(answer, result) {
		t.Fatalf("Expected %v, got %v", answer, result)
	}
}

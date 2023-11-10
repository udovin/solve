package db

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"
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

func (e *mockEvent) SetEventID(id int64) {
	e.ID = id
}

func (e mockEvent) EventTime() time.Time {
	if e.Time == 0 {
		return time.Now()
	}
	return time.Unix(e.Time, 0)
}

type mockEventStore struct {
	events []mockEvent
}

func (s *mockEventStore) LastEventID(ctx context.Context) (int64, error) {
	return 0, nil
}

type eventSorter []mockEvent

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
	ctx context.Context, ranges []EventRange,
) (Rows[mockEvent], error) {
	var events []mockEvent
	for _, rng := range ranges {
		for _, event := range s.events {
			if rng.contains(event.EventID()) {
				events = append(events, event)
			}
		}
	}
	sort.Sort(eventSorter(events))
	return NewSliceRows(events), nil
}

func (s *mockEventStore) FindEvents(
	tx *sql.Tx, where string, args ...any,
) (Rows[mockEvent], error) {
	return nil, sql.ErrNoRows
}

func TestEventConsumer(t *testing.T) {
	groups := [][]mockEvent{
		{
			{ID: 1}, {ID: 2}, {ID: 3},
		},
		{
			{ID: 5}, {ID: 6}, {ID: 8},
		},
		{
			{ID: 4}, {ID: 7}, {ID: 100},
		},
		{
			{ID: 50}, {ID: 75}, {ID: 101},
		},
		{
			{ID: 51}, {ID: 74}, {ID: 102},
		},
		{
			{ID: 25}, {ID: 97}, {ID: 98}, {ID: 99}, {ID: 103},
		},
		{
			{ID: 27}, {ID: 28}, {ID: 29}, {ID: 104},
		},
		{
			{ID: 26},
		},
	}
	store := &mockEventStore{}
	consumer := NewEventConsumer[mockEvent](store, 1)
	var result, answer []mockEvent
	usedIDs := map[int64]struct{}{}
	currID := int64(1)
	for _, group := range groups {
		for _, event := range group {
			store.events = append(store.events, event)
			answer = append(answer, event)
		}
		errConsume := fmt.Errorf("consuming error")
		if err := consumer.ConsumeEvents(context.Background(), func(event mockEvent) error {
			return errConsume
		}); err != errConsume {
			t.Fatal(err)
		}
		if err := consumer.ConsumeEvents(context.Background(), func(event mockEvent) error {
			result = append(result, event)
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

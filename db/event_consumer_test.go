package db

import (
	"container/list"
	"database/sql"
	"fmt"
	"reflect"
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

func (e mockEvent) EventTime() time.Time {
	if e.Time == 0 {
		return time.Now()
	}
	return time.Unix(e.Time, 0)
}

type mockEventStore struct {
	events *list.List
}

func (s *mockEventStore) LastEventID(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *mockEventStore) LoadEvents(
	tx *sql.Tx, begin, end int64,
) (EventReader, error) {
	lv := s.events.Front()
	if lv == nil {
		return nil, sql.ErrNoRows
	}
	events := list.New()
	l := lv.Value.(*list.List)
	for it := l.Front(); it != nil; {
		jt := it.Next()
		event := it.Value.(Event)
		if event.EventID() >= begin && event.EventID() < end {
			events.PushBack(event)
			l.Remove(it)
		}
		it = jt
	}
	if l.Len() == 0 {
		s.events.Remove(lv)
	}
	return &mockEventReader{events: events}, nil
}

func (s *mockEventStore) FindEvents(
	tx *sql.Tx, where string, args ...interface{},
) (EventReader, error) {
	return nil, sql.ErrNoRows
}

func newMockEventStore(groups [][]Event) *mockEventStore {
	store := mockEventStore{
		events: list.New(),
	}
	for _, group := range groups {
		events := list.New()
		for _, event := range group {
			events.PushBack(event)
		}
		store.events.PushBack(events)
	}
	return &store
}

type mockEventReader struct {
	events *list.List
	event  Event
}

func (r *mockEventReader) Next() bool {
	if it := r.events.Front(); it != nil {
		r.event = it.Value.(Event)
		r.events.Remove(it)
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
	store := newMockEventStore(groups)
	consumer := NewEventConsumer(store, 1)
	var result, answer []mockEvent
	usedIDs := map[int64]struct{}{}
	currID := int64(1)
	for _, group := range groups {
		for _, event := range group {
			answer = append(answer, event.(mockEvent))
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

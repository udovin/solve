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
	Id   int64 `db:"id"`
	Time int64 `db:"time"`
}

func (e mockEvent) String() string {
	return fmt.Sprintf("%d", e.Id)
}

func (e mockEvent) EventId() int64 {
	return e.Id
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
		if event.EventId() >= begin && event.EventId() < end {
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
			mockEvent{Id: 1}, mockEvent{Id: 2}, mockEvent{Id: 3},
		},
		{
			mockEvent{Id: 5}, mockEvent{Id: 6}, mockEvent{Id: 8},
		},
		{
			mockEvent{Id: 4}, mockEvent{Id: 7}, mockEvent{Id: 100},
		},
		{
			mockEvent{Id: 50}, mockEvent{Id: 75}, mockEvent{Id: 101},
		},
		{
			mockEvent{Id: 51}, mockEvent{Id: 74}, mockEvent{Id: 102},
		},
		{
			mockEvent{Id: 25}, mockEvent{Id: 97}, mockEvent{Id: 98},
			mockEvent{Id: 99}, mockEvent{Id: 103},
		},
		{
			mockEvent{Id: 27}, mockEvent{Id: 28}, mockEvent{Id: 29},
			mockEvent{Id: 104},
		},
		{
			mockEvent{Id: 26},
		},
	}
	store := newMockEventStore(groups)
	consumer := NewEventConsumer(store, 1)
	var result, answer []mockEvent
	usedIds := map[int64]struct{}{}
	currId := int64(1)
	for _, group := range groups {
		for _, event := range group {
			answer = append(answer, event.(mockEvent))
		}
		if err := consumer.ConsumeEvents(nil, func(event Event) error {
			result = append(result, event.(mockEvent))
			usedIds[event.EventId()] = struct{}{}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		for {
			if _, ok := usedIds[currId]; !ok {
				break
			}
			currId++
		}
		if consumer.BeginEventId() != currId {
			t.Fatalf("Expected %d, got %d", currId, consumer.BeginEventId())
		}
	}
	if !reflect.DeepEqual(answer, result) {
		t.Fatalf("Expected %v, got %v", answer, result)
	}
}

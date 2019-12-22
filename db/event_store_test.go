package db

import (
	"testing"
)

type testExtraEvent struct {
	A string `db:"a"`
	B int    `db:"b"`
}

type testEvent struct {
	mockEvent
	testExtraEvent
	C int `db:"c"`
}

func TestEventStore(t *testing.T) {
	_ = NewEventStore(testEvent{}, "test_event", "id")
}

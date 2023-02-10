package models

import (
	"github.com/udovin/gosql"
)

type ContestMessage struct {
	baseObject
	ContestID int64  `json:"contest_id"`
	ParentID  NInt64 `json:"parent_id"`
}

// Clone create copy of contest message.
func (o ContestMessage) Clone() ContestMessage {
	return o
}

type ContestMessageEvent struct {
	baseEvent
	ContestMessage
}

// Object returns event contest message.
func (e ContestMessageEvent) Object() ContestMessage {
	return e.ContestMessage
}

// SetObject sets event contest message.
func (e *ContestMessageEvent) SetObject(o ContestMessage) {
	e.ContestMessage = o
}

type ContestMessageStore struct {
	cachedStore[ContestMessage, ContestMessageEvent, *ContestMessage, *ContestMessageEvent]
}

func NewContestMessageStore(
	db *gosql.DB, table, eventTable string,
) *ContestMessageStore {
	impl := &ContestMessageStore{}
	impl.cachedStore = makeCachedStore[ContestMessage, ContestMessageEvent](
		db, table, eventTable, impl,
	)
	return impl
}

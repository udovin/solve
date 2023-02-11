package models

import (
	"github.com/udovin/gosql"
)

type ContestMessageKind int

const (
	RegularContestMessage  ContestMessageKind = 0
	QuestionContestMessage ContestMessageKind = 1
	AnswerContestMessage   ContestMessageKind = 2
)

type ContestMessage struct {
	baseObject
	ContestID     int64              `json:"contest_id"`
	ParticipantID NInt64             `json:"participant_id"`
	AuthorID      int64              `json:"author_id"`
	Kind          ContestMessageKind `json:"kind"`
	ParentID      NInt64             `json:"parent_id"`
	Title         string             `json:"title"`
	Description   string             `json:"description"`
	CreateTime    int64              `json:"create_time"`
	ProblemID     NInt64             `json:"problem_id"`
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

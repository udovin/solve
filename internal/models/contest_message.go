package models

import (
	"context"
	"fmt"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
)

type ContestMessageKind int

const (
	RegularContestMessage  ContestMessageKind = 0
	QuestionContestMessage ContestMessageKind = 1
	AnswerContestMessage   ContestMessageKind = 2
)

func (k ContestMessageKind) String() string {
	switch k {
	case RegularContestMessage:
		return "regular"
	case QuestionContestMessage:
		return "question"
	case AnswerContestMessage:
		return "answer"
	default:
		return fmt.Sprintf("ContestMessageKind(%d)", k)
	}
}

type ContestMessage struct {
	baseObject
	ContestID     int64              `db:"contest_id"`
	ParticipantID NInt64             `db:"participant_id"`
	AuthorID      int64              `db:"author_id"`
	Kind          ContestMessageKind `db:"kind"`
	ParentID      NInt64             `db:"parent_id"`
	Title         string             `db:"title"`
	Description   string             `db:"description"`
	CreateTime    int64              `db:"create_time"`
	ProblemID     NInt64             `db:"problem_id"`
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

type ContestMessageStore interface {
	Store[ContestMessage, ContestMessageEvent]
	FindByContest(ctx context.Context, contestID int64) (db.Rows[ContestMessage], error)
}

type cachedContestMessageStore struct {
	cachedStore[ContestMessage, ContestMessageEvent, *ContestMessage, *ContestMessageEvent]
	byContest *btreeIndex[int64, ContestMessage, *ContestMessage]
}

func (s *cachedContestMessageStore) FindByContest(
	ctx context.Context, contestID int64,
) (db.Rows[ContestMessage], error) {
	s.mutex.RLock()
	return btreeIndexFind(
		s.byContest,
		s.objects.Iter(),
		s.mutex.RLocker(),
		contestID,
	), nil
}

func NewCachedContestMessageStore(
	db *gosql.DB, table, eventTable string,
) ContestMessageStore {
	impl := &cachedContestMessageStore{
		byContest: newBTreeIndex(func(o ContestMessage) (int64, bool) { return o.ContestID, true }, lessInt64),
	}
	impl.cachedStore = makeCachedStore[ContestMessage, ContestMessageEvent](
		db, table, eventTable, impl, impl.byContest,
	)
	return impl
}

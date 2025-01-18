package models

import (
	"context"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
)

type ContestFakeParticipant struct {
	ID        int64  `db:"id"`
	ContestID int64  `db:"contest_id"`
	Title     string `db:"title"`
}

func (o ContestFakeParticipant) ObjectID() int64 {
	return o.ID
}

func (o *ContestFakeParticipant) SetObjectID(id int64) {
	o.ID = id
}

type ContestFakeParticipantStore struct {
	store db.ObjectStore[ContestFakeParticipant, *ContestFakeParticipant]
}

func (s *ContestFakeParticipantStore) Create(ctx context.Context, object *ContestFakeParticipant) error {
	return s.store.CreateObject(ctx, object)
}

func (s *ContestFakeParticipantStore) Update(ctx context.Context, object ContestFakeParticipant) error {
	return s.store.UpdateObject(ctx, &object)
}

func (s *ContestFakeParticipantStore) Delete(ctx context.Context, id int64) error {
	return s.store.DeleteObject(ctx, id)
}

func (s *ContestFakeParticipantStore) Get(ctx context.Context, id int64) (ContestFakeParticipant, error) {
	return s.store.FindObject(ctx, db.FindQuery{Where: gosql.Column("id").Equal(id)})
}

func (s *ContestFakeParticipantStore) FindByContest(
	ctx context.Context, contestID int64,
) (db.Rows[ContestFakeParticipant], error) {
	return s.store.FindObjects(ctx, db.FindQuery{Where: gosql.Column("contest_id").Equal(contestID)})
}

func NewContestFakeParticipantStore(conn *gosql.DB, table string) *ContestFakeParticipantStore {
	impl := &ContestFakeParticipantStore{
		store: db.NewObjectStore[ContestFakeParticipant, *ContestFakeParticipant]("id", table, conn),
	}
	return impl
}

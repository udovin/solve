package models

import (
	"context"
	"encoding/json"

	"github.com/udovin/gosql"
	"github.com/udovin/solve/internal/db"
)

type FakeSolutionReport struct {
	Verdict Verdict  `json:"verdict"`
	Points  *float64 `json:"points,omitempty"`
}

type ContestFakeSolution struct {
	ID            int64 `db:"id"`
	ContestID     int64 `db:"contest_id"`
	ParticipantID int64 `db:"participant_id"`
	ProblemID     int64 `db:"problem_id"`
	ContestTime   int64 `db:"contest_time"`
	Report        JSON  `db:"report"`
}

func (o ContestFakeSolution) ObjectID() int64 {
	return o.ID
}

func (o *ContestFakeSolution) SetObjectID(id int64) {
	o.ID = id
}

// GetReport returns solution report.
func (o ContestFakeSolution) GetReport() (*FakeSolutionReport, error) {
	if o.Report == nil {
		return nil, nil
	}
	var report *FakeSolutionReport
	err := json.Unmarshal(o.Report, &report)
	return report, err
}

// SetReport sets serialized report to solution.
func (o *ContestFakeSolution) SetReport(report *FakeSolutionReport) error {
	if report == nil {
		o.Report = nil
		return nil
	}
	raw, err := json.Marshal(report)
	if err != nil {
		return err
	}
	o.Report = raw
	return nil
}

// Clone creates copy of contest problem.
func (o ContestFakeSolution) Clone() ContestFakeSolution {
	o.Report = o.Report.Clone()
	return o
}

type ContestFakeSolutionStore struct {
	store db.ObjectStore[ContestFakeSolution, *ContestFakeSolution]
}

func (s *ContestFakeSolutionStore) Create(ctx context.Context, object *ContestFakeSolution) error {
	return s.store.CreateObject(ctx, object)
}

func (s *ContestFakeSolutionStore) Update(ctx context.Context, object ContestFakeSolution) error {
	return s.store.UpdateObject(ctx, &object)
}

func (s *ContestFakeSolutionStore) Delete(ctx context.Context, id int64) error {
	return s.store.DeleteObject(ctx, id)
}

func (s *ContestFakeSolutionStore) Get(ctx context.Context, id int64) (ContestFakeSolution, error) {
	return s.store.FindObject(ctx, db.FindQuery{Where: gosql.Column("id").Equal(id)})
}

func (s *ContestFakeSolutionStore) FindByContest(
	ctx context.Context, contestID int64,
) (db.Rows[ContestFakeSolution], error) {
	return s.store.FindObjects(ctx, db.FindQuery{Where: gosql.Column("contest_id").Equal(contestID)})
}

func NewContestFakeSolutionStore(conn *gosql.DB, table string) *ContestFakeSolutionStore {
	impl := &ContestFakeSolutionStore{
		store: db.NewObjectStore[ContestFakeSolution, *ContestFakeSolution]("id", table, conn),
	}
	return impl
}

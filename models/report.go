package models

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

const (
	// Accepted means that report is correct
	Accepted = 1
	// CompilationError means that report can not compiled
	CompilationError = 2
	// TimeLimitExceeded means that report uses more time than allowed
	TimeLimitExceeded = 3
	// MemoryLimitExceeded means that report uses more memory than allowed
	MemoryLimitExceeded = 4
	// RuntimeError means that report runs incorrectly
	RuntimeError = 5
	// WrongAnswer means that report is incorrect
	WrongAnswer = 6
	// PresentationError means that report output is incorrect
	PresentationError = 7
)

type Report struct {
	ID         int64  `json:"" db:"id"`
	SolutionID int64  `json:"" db:"solution_id"`
	Verdict    int8   `json:"" db:"verdict"`
	Data       string `json:"" db:"data"`
	CreateTime int64  `json:"" db:"create_time"`
}

type reportChange struct {
	BaseChange
	Report
}

type ReportStore struct {
	Manager     *ChangeManager
	table       string
	changeTable string
	reports     map[int64]Report
	queued      map[int64]struct{}
	mutex       sync.RWMutex
}

func NewReportStore(db *sql.DB, table, changeTable string) *ReportStore {
	store := ReportStore{
		table:       table,
		changeTable: changeTable,
		reports:     make(map[int64]Report),
		queued:      make(map[int64]struct{}),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *ReportStore) Get(id int64) (Report, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	report, ok := s.reports[id]
	return report, ok
}

func (s *ReportStore) GetQueuedIDs() (ids []int64) {
	for id := range s.queued {
		ids = append(ids, id)
	}
	return
}

func (s *ReportStore) Create(m *Report) error {
	change := reportChange{
		BaseChange: BaseChange{Type: CreateChange},
		Report:     *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Report
	return nil
}

func (s *ReportStore) Update(m *Report) error {
	change := reportChange{
		BaseChange: BaseChange{Type: UpdateChange},
		Report:     *m,
	}
	err := s.Manager.Change(&change)
	if err != nil {
		return err
	}
	*m = change.Report
	return nil
}

func (s *ReportStore) UpdateTx(tx *ChangeTx, m *Report) error {
	change := reportChange{
		BaseChange: BaseChange{Type: UpdateChange},
		Report:     *m,
	}
	err := s.Manager.ChangeTx(tx, &change)
	if err != nil {
		return err
	}
	*m = change.Report
	return nil
}

func (s *ReportStore) Delete(id int64) error {
	change := reportChange{
		BaseChange: BaseChange{Type: DeleteChange},
		Report:     Report{ID: id},
	}
	return s.Manager.Change(&change)
}

func (s *ReportStore) GetLocker() sync.Locker {
	return &s.mutex
}

func (s *ReportStore) InitChanges(tx *sql.Tx) (int64, error) {
	return 0, nil
}

func (s *ReportStore) LoadChanges(
	tx *sql.Tx, gap ChangeGap,
) (*sql.Rows, error) {
	return tx.Query(
		fmt.Sprintf(
			`SELECT`+
				` "change_id", "change_type", "change_time",`+
				` "id", "solution_id", "verdict", "data", "create_time"`+
				` FROM "%s"`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *ReportStore) ScanChange(scan Scanner) (Change, error) {
	report := reportChange{}
	err := scan.Scan(
		&report.BaseChange.ID, &report.Type, &report.Time,
		&report.Report.ID, &report.SolutionID, &report.Verdict,
		&report.Data, &report.CreateTime,
	)
	return &report, err
}

func (s *ReportStore) SaveChange(tx *sql.Tx, change Change) error {
	report := change.(*reportChange)
	report.Time = time.Now().Unix()
	switch report.Type {
	case CreateChange:
		report.Report.CreateTime = report.Time
		res, err := tx.Exec(
			fmt.Sprintf(
				`INSERT INTO "%s"`+
					` ("solution_id", "verdict", "data", "create_time")`+
					` VALUES ($1, $2, $3, $4)`,
				s.table,
			),
			report.SolutionID, report.Verdict,
			report.Data, report.CreateTime,
		)
		if err != nil {
			return err
		}
		report.Report.ID, err = res.LastInsertId()
		if err != nil {
			return err
		}
	case UpdateChange:
		if _, ok := s.reports[report.Report.ID]; !ok {
			return fmt.Errorf(
				"report with id = %d does not exists",
				report.Report.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`UPDATE "%s"`+
					` SET "solution_id" = $1, "verdict" = $2, "data" = $3`+
					` WHERE "id" = $5`,
				s.table,
			),
			report.SolutionID, report.Verdict,
			report.Data, report.Report.ID,
		)
		if err != nil {
			return err
		}
	case DeleteChange:
		if _, ok := s.reports[report.Report.ID]; !ok {
			return fmt.Errorf(
				"report with id = %d does not exists",
				report.Report.ID,
			)
		}
		_, err := tx.Exec(
			fmt.Sprintf(
				`DELETE FROM "%s" WHERE "id" = $1`,
				s.table,
			),
			report.Report.ID,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"unsupported change type = %s",
			report.Type,
		)
	}
	res, err := tx.Exec(
		fmt.Sprintf(
			`INSERT INTO "%s"`+
				` ("change_type", "change_time",`+
				` "id", "solution_id", "verdict",`+
				` "data", "create_time")`+
				` VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			s.changeTable,
		),
		report.Type, report.Time,
		report.Report.ID, report.SolutionID, report.Verdict,
		report.Data, report.CreateTime,
	)
	if err != nil {
		return err
	}
	report.BaseChange.ID, err = res.LastInsertId()
	return err
}

func (s *ReportStore) ApplyChange(change Change) {
	report := change.(*reportChange)
	switch report.Type {
	case UpdateChange:
		if report.Verdict != 0 {
			delete(s.queued, report.Report.ID)
		}
		fallthrough
	case CreateChange:
		s.reports[report.Report.ID] = report.Report
		if report.Verdict == 0 {
			s.queued[report.Report.ID] = struct{}{}
		}
	case DeleteChange:
		delete(s.reports, report.Report.ID)
		delete(s.queued, report.Report.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			report.Type,
		))
	}
}

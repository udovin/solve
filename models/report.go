package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
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

type ReportDataLogs struct {
	Stderr string `json:""`
	Stdout string `json:""`
}

type ReportDataUsage struct {
	Time   int64 `json:""`
	Memory int64 `json:""`
}

type ReportDataTest struct {
	CheckLogs ReportDataLogs  `json:""`
	Usage     ReportDataUsage `json:""`
	Verdict   int8            `json:""`
	Points    *float64        `json:""`
}

// TODO(iudovin): remove 'Defense'
type ReportData struct {
	PrecompileLogs ReportDataLogs   `json:""`
	CompileLogs    ReportDataLogs   `json:""`
	Usage          ReportDataUsage  `json:""`
	Tests          []ReportDataTest `json:""`
	Points         *float64         `json:""`
	Defense        *int8            `json:""`
}

func (d ReportData) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d *ReportData) Scan(value interface{}) error {
	switch data := value.(type) {
	case []byte:
		return json.Unmarshal(data, d)
	case string:
		return json.Unmarshal([]byte(data), d)
	default:
		return fmt.Errorf("unsupported type: %T", data)
	}
}

type Report struct {
	ID         int64      `json:"" db:"id"`
	SolutionID int64      `json:"" db:"solution_id"`
	Verdict    int8       `json:"" db:"verdict"`
	Data       ReportData `json:"" db:"data"`
	CreateTime int64      `json:"" db:"create_time"`
}

type ReportChange struct {
	BaseChange
	Report
}

type ReportStore struct {
	Manager         *ChangeManager
	table           string
	changeTable     string
	reports         map[int64]Report
	solutionReports map[int64]map[int64]struct{}
	queued          map[int64]struct{}
	mutex           sync.RWMutex
}

func NewReportStore(db *sql.DB, table, changeTable string) *ReportStore {
	store := ReportStore{
		table:           table,
		changeTable:     changeTable,
		reports:         make(map[int64]Report),
		solutionReports: make(map[int64]map[int64]struct{}),
		queued:          make(map[int64]struct{}),
	}
	store.Manager = NewChangeManager(&store, db)
	return &store
}

func (s *ReportStore) Get(id int64) (Report, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if report, ok := s.reports[id]; ok {
		return report, nil
	}
	return Report{}, sql.ErrNoRows
}

func (s *ReportStore) GetLatest(solutionID int64) (Report, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if ids, ok := s.solutionReports[solutionID]; ok {
		var latest Report
		for id := range ids {
			if report, ok := s.reports[id]; ok {
				if report.ID > latest.ID {
					latest = report
				}
			}
		}
		return latest, nil
	}
	return Report{}, sql.ErrNoRows
}

func (s *ReportStore) GetQueuedIDs() (ids []int64) {
	for id := range s.queued {
		ids = append(ids, id)
	}
	return
}

func (s *ReportStore) Create(m *Report) error {
	change := ReportChange{
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

func (s *ReportStore) CreateTx(tx *ChangeTx, m *Report) error {
	change := ReportChange{
		BaseChange: BaseChange{Type: CreateChange},
		Report:     *m,
	}
	err := s.Manager.ChangeTx(tx, &change)
	if err != nil {
		return err
	}
	*m = change.Report
	return nil
}

func (s *ReportStore) Update(m *Report) error {
	change := ReportChange{
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
	change := ReportChange{
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
	change := ReportChange{
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
				` FROM %q`+
				` WHERE "change_id" >= $1 AND "change_id" < $2`+
				` ORDER BY "change_id"`,
			s.changeTable,
		),
		gap.BeginID, gap.EndID,
	)
}

func (s *ReportStore) ScanChange(scan Scanner) (Change, error) {
	report := ReportChange{}
	err := scan.Scan(
		&report.BaseChange.ID, &report.Type, &report.Time,
		&report.Report.ID, &report.SolutionID, &report.Verdict,
		&report.Data, &report.CreateTime,
	)
	return &report, err
}

func (s *ReportStore) SaveChange(tx *sql.Tx, change Change) error {
	report := change.(*ReportChange)
	report.Time = time.Now().Unix()
	switch report.Type {
	case CreateChange:
		report.Report.CreateTime = report.Time
		var err error
		report.Report.ID, err = execTxReturningID(
			s.Manager.db.Driver(), tx,
			fmt.Sprintf(
				`INSERT INTO %q`+
					` ("solution_id", "verdict", "data", "create_time")`+
					` VALUES ($1, $2, $3, $4)`,
				s.table,
			),
			"id",
			report.SolutionID, report.Verdict,
			report.Data, report.CreateTime,
		)
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
				`UPDATE %q`+
					` SET "solution_id" = $1, "verdict" = $2, "data" = $3`+
					` WHERE "id" = $4`,
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
				`DELETE FROM %q WHERE "id" = $1`,
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
	var err error
	report.BaseChange.ID, err = execTxReturningID(
		s.Manager.db.Driver(), tx,
		fmt.Sprintf(
			`INSERT INTO %q`+
				` ("change_type", "change_time",`+
				` "id", "solution_id", "verdict",`+
				` "data", "create_time")`+
				` VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			s.changeTable,
		),
		"change_id",
		report.Type, report.Time,
		report.Report.ID, report.SolutionID, report.Verdict,
		report.Data, report.CreateTime,
	)
	return err
}

func (s *ReportStore) ApplyChange(change Change) {
	report := change.(*ReportChange)
	switch report.Type {
	case UpdateChange:
		if old, ok := s.reports[report.Report.ID]; ok {
			if old.SolutionID != old.SolutionID {
				if reports, ok := s.solutionReports[old.SolutionID]; ok {
					delete(reports, old.ID)
					if len(reports) == 0 {
						delete(s.solutionReports, old.SolutionID)
					}
				}
			}
		}
		if report.Verdict != 0 {
			delete(s.queued, report.Report.ID)
		}
		fallthrough
	case CreateChange:
		if _, ok := s.solutionReports[report.SolutionID]; !ok {
			s.solutionReports[report.SolutionID] = make(map[int64]struct{})
		}
		s.solutionReports[report.SolutionID][report.Report.ID] = struct{}{}
		if report.Verdict == 0 {
			s.queued[report.Report.ID] = struct{}{}
		}
		s.reports[report.Report.ID] = report.Report
	case DeleteChange:
		if reports, ok := s.solutionReports[report.SolutionID]; ok {
			delete(reports, report.Report.ID)
			if len(reports) == 0 {
				delete(s.solutionReports, report.SolutionID)
			}
		}
		delete(s.reports, report.Report.ID)
		delete(s.queued, report.Report.ID)
	default:
		panic(fmt.Errorf(
			"unsupported change type = %s",
			report.Type,
		))
	}
}

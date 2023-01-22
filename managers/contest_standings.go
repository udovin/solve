package managers

import (
	"context"
	"database/sql"
	"sort"

	"github.com/udovin/solve/core"
	"github.com/udovin/solve/models"
)

type ContestStandingsManager struct {
	contestParticipants *models.ContestParticipantStore
	contestSolutions    *models.ContestSolutionStore
	contestProblems     *models.ContestProblemStore
	solutions           *models.SolutionStore
}

func NewContestStandingsManager(core *core.Core) *ContestStandingsManager {
	return &ContestStandingsManager{
		contestParticipants: core.ContestParticipants,
		contestSolutions:    core.ContestSolutions,
		contestProblems:     core.ContestProblems,
		solutions:           core.Solutions,
	}
}

func (m *ContestStandingsManager) BuildStandings(ctx context.Context, contest models.Contest) (*ContestStandings, error) {
	contestConfig, err := contest.GetConfig()
	if err != nil {
		return nil, err
	}
	participants, err := m.contestParticipants.FindByContest(contest.ID)
	if err != nil {
		return nil, err
	}
	contestProblems, err := m.contestProblems.FindByContest(contest.ID)
	if err != nil {
		return nil, err
	}
	sortFunc(contestProblems, func(lhs, rhs models.ContestProblem) bool {
		return lhs.Code < rhs.Code
	})
	var columns []ContestStandingsColumn
	columnByProblem := map[int64]int{}
	for i, problem := range contestProblems {
		columns = append(columns, ContestStandingsColumn{
			Problem: problem,
		})
		columnByProblem[problem.ID] = i
	}
	contestSolutions, err := m.contestSolutions.FindByContest(contest.ID)
	if err != nil {
		return nil, err
	}
	solutionsByParticipant := map[int64][]models.ContestSolution{}
	for _, solution := range contestSolutions {
		solutionsByParticipant[solution.ParticipantID] = append(
			solutionsByParticipant[solution.ParticipantID], solution,
		)
	}
	standings := ContestStandings{}
	for _, participant := range participants {
		beginTime := int64(contestConfig.BeginTime)
		if participant.Kind == models.RegularParticipant {
			var participantConfig models.RegularParticipantConfig
			if err := participant.ScanConfig(&participantConfig); err != nil {
				continue
			}
			if participantConfig.BeginTime != 0 {
				beginTime = int64(participantConfig.BeginTime)
			}
		}
		participantSolutions, ok := solutionsByParticipant[participant.ID]
		if !ok {
			continue
		}
		sortFunc(participantSolutions, func(lhs, rhs models.ContestSolution) bool {
			return lhs.ID < rhs.ID
		})
		solutionsByColumn := map[int][]models.Solution{}
		for _, participantSolution := range participantSolutions {
			solution, err := m.solutions.Get(participantSolution.SolutionID)
			if err != nil {
				if err == sql.ErrNoRows {
					continue
				}
				return nil, err
			}
			column, ok := columnByProblem[participantSolution.ProblemID]
			if !ok {
				continue
			}
			solutionsByColumn[column] = append(solutionsByColumn[column], solution)
		}
		row := ContestStandingsRow{
			Participant: participant,
		}
		for i := range columns {
			solutions, ok := solutionsByColumn[i]
			if !ok {
				continue
			}
			cell := ContestStandingsCell{
				Column: i,
			}
			for _, solution := range solutions {
				report, err := solution.GetReport()
				if err != nil {
					continue
				}
				if report == nil {
					cell.Attempt++
					cell.Verdict = 0
					break
				}
				if report.Verdict == models.CompilationError {
					continue
				}
				cell.Attempt++
				cell.Verdict = report.Verdict
				if report.Verdict == models.Accepted {
					cell.Time = solution.CreateTime - beginTime
					if cell.Time < 0 {
						cell.Time = 0
					}
					break
				}
			}
			row.Cells = append(row.Cells, cell)
		}
		for _, cell := range row.Cells {
			if cell.Verdict == models.Accepted {
				row.Score++
				row.Penalty = int64(cell.Attempt-1)*20 + cell.Time/60
			}
		}
		standings.Rows = append(standings.Rows, row)
	}
	return &standings, nil
}

type ContestStandingsColumn struct {
	Problem models.ContestProblem
}

type ContestStandingsCell struct {
	Column  int
	Verdict models.Verdict
	Attempt int
	Time    int64
}

type ContestStandingsRow struct {
	Participant models.ContestParticipant
	Cells       []ContestStandingsCell
	Score       int
	Penalty     int64
}

type ContestStandings struct {
	Columns []ContestStandingsColumn
	Rows    []ContestStandingsRow
}

func sortFunc[T any](a []T, less func(T, T) bool) {
	impl := sortFuncImpl[T]{data: a, less: less}
	sort.Sort(&impl)
}

type sortFuncImpl[T any] struct {
	data []T
	less func(T, T) bool
}

func (s *sortFuncImpl[T]) Len() int {
	return len(s.data)
}

func (s *sortFuncImpl[T]) Swap(i, j int) {
	s.data[i], s.data[j] = s.data[j], s.data[i]
}

func (s *sortFuncImpl[T]) Less(i, j int) bool {
	return s.less(s.data[i], s.data[j])
}

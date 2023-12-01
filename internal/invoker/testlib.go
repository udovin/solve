package invoker

import (
	"context"
	"fmt"
	"time"

	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/compilers"
)

func getTestlibExitCodeVerdict(exitCode int) (models.Verdict, error) {
	switch exitCode {
	case 0:
		return models.Accepted, nil
	case 1:
		return models.WrongAnswer, nil
	case 3:
		return models.Failed, nil
	case 2, 4, 8:
		return models.PresentationError, nil
	case 5:
		return models.PartiallyAccepted, nil
	default:
		if exitCode < 16 {
			return 0, fmt.Errorf("unknown exit code: %d", exitCode)
		}
		return models.PartiallyAccepted, nil
	}
}

func runTestlibChecker(ctx context.Context, checker compilers.Executable, inputPath, outputPath, answerPath string) (models.TestReport, error) {
	log := truncateBuffer{limit: 2048}
	process, err := checker.CreateProcess(ctx, compilers.ExecuteOptions{
		Args:        []string{"input.in", "output.out", "answer.ans"},
		Stderr:      &log,
		TimeLimit:   20 * time.Second,
		MemoryLimit: 256 * 1024 * 1024,
	})
	if err != nil {
		return models.TestReport{}, fmt.Errorf("cannot create checker process: %w", err)
	}
	defer func() { _ = process.Release() }()
	if err := copyFileRec(inputPath, process.UpperPath("input.in")); err != nil {
		return models.TestReport{}, fmt.Errorf("cannot write checker input file: %w", err)
	}
	if err := copyFileRec(outputPath, process.UpperPath("output.out")); err != nil {
		return models.TestReport{}, fmt.Errorf("cannot write checker output file: %w", err)
	}
	if err := copyFileRec(answerPath, process.UpperPath("answer.ans")); err != nil {
		return models.TestReport{}, fmt.Errorf("cannot write checker answer file: %w", err)
	}
	if err := process.Start(); err != nil {
		return models.TestReport{}, fmt.Errorf("cannot start checker: %w", err)
	}
	report, err := process.Wait()
	if err != nil {
		return models.TestReport{}, fmt.Errorf("cannot wait checker: %w", err)
	}
	verdict, err := getTestlibExitCodeVerdict(report.ExitCode)
	if err != nil {
		return models.TestReport{}, fmt.Errorf("checker returned error: %w", err)
	}
	return models.TestReport{
		Verdict: verdict,
		Checker: &models.ExecuteReport{
			Usage: models.UsageReport{
				Time:   report.Time.Milliseconds(),
				Memory: report.Memory,
			},
			Log: log.String(),
		},
	}, nil
}

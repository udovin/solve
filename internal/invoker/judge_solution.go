package invoker

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/udovin/algo/futures"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/compilers"
	"github.com/udovin/solve/internal/pkg/logs"
	"github.com/udovin/solve/internal/pkg/problems"
	"github.com/udovin/solve/internal/pkg/safeexec"
)

func init() {
	registerTaskImpl(models.JudgeSolutionTask, &judgeSolutionTask{})
}

type judgeSolutionTask struct {
	invoker        *Invoker
	config         models.JudgeSolutionTaskConfig
	solution       models.Solution
	problem        models.Problem
	tempDir        string
	problemImpl    problems.Problem
	compiler       compilers.Compiler
	solutionImpl   compilers.Executable
	interactorImpl compilers.Executable
	checkerImpl    compilers.Executable
	solutionPath   string
	compiledPath   string
}

func (judgeSolutionTask) New(invoker *Invoker) taskImpl {
	return &judgeSolutionTask{invoker: invoker}
}

func (t *judgeSolutionTask) Execute(ctx TaskContext) error {
	// Fetch information about task.
	if err := ctx.ScanConfig(&t.config); err != nil {
		return fmt.Errorf("unable to scan task config: %w", err)
	}
	syncCtx := models.WithSync(ctx)
	solution, err := t.invoker.core.Solutions.Get(syncCtx, t.config.SolutionID)
	if err != nil {
		return fmt.Errorf("unable to fetch solution: %w", err)
	}
	compileCtx := t.newCompileContext(ctx)
	defer compileCtx.Release()
	problem, err := t.invoker.core.Problems.Get(syncCtx, solution.ProblemID)
	if err != nil {
		return fmt.Errorf("unable to fetch problem: %w", err)
	}
	compiler, err := compileCtx.GetCompilerByID(ctx, solution.CompilerID)
	if err != nil {
		return fmt.Errorf("unable to fetch compiler: %w", err)
	}
	tempDir, err := makeTempDir()
	if err != nil {
		return err
	}
	defer func() {
		if t.checkerImpl != nil {
			t.checkerImpl.Release()
		}
		if t.solutionImpl != nil {
			t.solutionImpl.Release()
		}
		if t.interactorImpl != nil {
			t.interactorImpl.Release()
		}
	}()
	defer func() { _ = os.RemoveAll(tempDir) }()
	t.tempDir = tempDir
	t.solution = solution
	t.problem = problem
	t.compiler = compiler
	return t.executeImpl(ctx, compileCtx)
}

func (t *judgeSolutionTask) newCompileContext(ctx TaskContext) *compileContext {
	return &compileContext{
		compilers: t.invoker.core.Compilers,
		cache:     t.invoker.compilerImages,
		logger:    ctx.Logger(),
	}
}

func (t *judgeSolutionTask) prepareProblem(ctx TaskContext) error {
	if t.problem.PackageID == 0 {
		return fmt.Errorf("problem does not have package")
	}
	problem, err := t.invoker.problems.DownloadProblem(
		ctx, t.problem, problems.CompiledProblem,
	)
	if err != nil {
		return fmt.Errorf("cannot download problem: %w", err)
	}
	t.problemImpl = problem
	return nil
}

func (t *judgeSolutionTask) prepareSolution(ctx TaskContext) error {
	if t.solution.ContentID == 0 {
		tempSolutionPath := filepath.Join(t.tempDir, "solution.txt")
		err := os.WriteFile(tempSolutionPath, []byte(t.solution.Content), fs.ModePerm)
		if err != nil {
			return fmt.Errorf("cannot write solution: %w", err)
		}
		t.solutionPath = tempSolutionPath
		return nil
	}
	solutionFile, err := t.invoker.files.DownloadFile(ctx, int64(t.solution.ContentID))
	if err != nil {
		return fmt.Errorf("cannot download solution: %w", err)
	}
	defer func() { _ = solutionFile.Close() }()
	tempSolutionPath := filepath.Join(t.tempDir, "solution.bin")
	file, err := os.Create(tempSolutionPath)
	if err != nil {
		return fmt.Errorf("cannot create solution: %w", err)
	}
	defer func() { _ = file.Close() }()
	if _, err := io.Copy(file, solutionFile); err != nil {
		return fmt.Errorf("cannot write solution: %w", err)
	}
	t.solutionPath = tempSolutionPath
	t.compiledPath = filepath.Join(t.tempDir, "solution")
	return nil
}

func (t *judgeSolutionTask) compileSolution(
	ctx TaskContext, report *models.SolutionReport,
) (bool, error) {
	state := models.JudgeSolutionTaskState{
		Stage: "compiling",
	}
	if err := ctx.SetDeferredState(state); err != nil {
		return false, err
	}
	compileReport, err := t.compiler.Compile(ctx, compilers.CompileOptions{
		Source:      t.solutionPath,
		Target:      t.compiledPath,
		TimeLimit:   20 * time.Second,
		MemoryLimit: 256 * 1024 * 1024,
	})
	if err != nil {
		return false, err
	}
	report.Compiler = &models.ExecuteReport{
		Log: compileReport.Log,
		Usage: models.UsageReport{
			Time:   compileReport.UsedTime.Milliseconds(),
			Memory: compileReport.UsedMemory,
		},
	}
	if !compileReport.Success() {
		return false, nil
	}
	exe, err := t.compiler.CreateExecutable(ctx, t.compiledPath)
	if err != nil {
		return false, err
	}
	t.solutionImpl = exe
	return true, nil
}

var (
	errNoChecker    = fmt.Errorf("cannot find checker executable")
	errNoInteractor = fmt.Errorf("cannot find interactor executable")
)

func (t *judgeSolutionTask) getChecker(
	ctx TaskContext, compileCtx problems.CompileContext, executables []problems.ProblemExecutable,
) (compilers.Executable, error) {
	var checker problems.ProblemExecutable
	for _, executable := range executables {
		if executable.Kind() == problems.TestlibChecker {
			checker = executable
			break
		}
	}
	if checker == nil {
		return nil, errNoChecker
	}
	compiler, err := checker.GetCompiler(ctx, compileCtx)
	if err != nil {
		return nil, err
	}
	checkerPath := filepath.Join(t.tempDir, "checker")
	if err := func() error {
		testFile, err := checker.OpenBinary()
		if err != nil {
			return err
		}
		file, err := os.OpenFile(checkerPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		if _, err := io.Copy(file, testFile); err != nil {
			return err
		}
		return file.Sync()
	}(); err != nil {
		return nil, err
	}
	return compiler.CreateExecutable(ctx, checkerPath)
}

func (t *judgeSolutionTask) getInteractor(
	ctx TaskContext, compileCtx problems.CompileContext, executables []problems.ProblemExecutable,
) (compilers.Executable, error) {
	var interactor problems.ProblemExecutable
	for _, executable := range executables {
		if executable.Kind() == problems.TestlibInteractor {
			interactor = executable
			break
		}
	}
	if interactor == nil {
		return nil, errNoInteractor
	}
	compiler, err := interactor.GetCompiler(ctx, compileCtx)
	if err != nil {
		return nil, err
	}
	interactorPath := filepath.Join(t.tempDir, "interactor")
	if err := func() error {
		testFile, err := interactor.OpenBinary()
		if err != nil {
			return err
		}
		file, err := os.OpenFile(interactorPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		if _, err := io.Copy(file, testFile); err != nil {
			return err
		}
		return file.Sync()
	}(); err != nil {
		return nil, err
	}
	return compiler.CreateExecutable(ctx, interactorPath)
}

func (t *judgeSolutionTask) calculateTestSetPoints(
	ctx TaskContext,
	report *models.SolutionReport,
	testSet problems.ProblemTestSet,
	groupTests map[string][]int,
) error {
	groups, err := testSet.GetGroups()
	if err != nil {
		return err
	}
	for _, group := range groups {
		groupPoints := float64(0)
		groupVerdict := models.Accepted
		for _, id := range groupTests[group.Name()] {
			test := report.Tests[id]
			if test.Points != nil {
				groupPoints += *test.Points
			}
			if test.Verdict != models.Accepted {
				groupVerdict = test.Verdict
			}
		}
		switch group.PointsPolicy() {
		case problems.EachTestPointsPolicy:
			*report.Points += groupPoints
		case problems.CompleteGroupPointsPolicy:
			if groupVerdict == models.Accepted {
				*report.Points += groupPoints
			} else {
				for _, id := range groupTests[group.Name()] {
					report.Tests[id].Points = nil
				}
			}
		default:
			return fmt.Errorf("unsupported policy: %v", group.PointsPolicy())
		}
	}
	return nil
}

func (t *judgeSolutionTask) prepareExecutables(ctx TaskContext, compileCtx problems.CompileContext) error {
	executables, err := t.problemImpl.GetExecutables()
	if err != nil {
		return fmt.Errorf("cannot get executables: %w", err)
	}
	t.checkerImpl, err = t.getChecker(ctx, compileCtx, executables)
	if err != nil {
		return err
	}
	t.interactorImpl, err = t.getInteractor(ctx, compileCtx, executables)
	if err != nil && err != errNoInteractor {
		return err
	}
	return nil
}

func (t *judgeSolutionTask) executeSolution(
	ctx context.Context,
	testSet problems.ProblemTestSet,
	inputPath, outputPath, answerPath string,
) (models.TestReport, error) {
	if t.interactorImpl != nil {
		return t.executeInteractiveSolution(ctx, testSet, inputPath, outputPath, answerPath)
	}
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return models.TestReport{}, err
	}
	defer func() { _ = inputFile.Close() }()
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return models.TestReport{}, err
	}
	process, err := t.solutionImpl.CreateProcess(ctx, compilers.ExecuteOptions{
		Stdin:       inputFile,
		Stdout:      outputFile,
		TimeLimit:   time.Duration(testSet.TimeLimit()) * time.Millisecond,
		MemoryLimit: testSet.MemoryLimit(),
	})
	if err != nil {
		return models.TestReport{}, fmt.Errorf("cannot prepare solution: %w", err)
	}
	defer func() { _ = process.Release() }()
	if err := process.Start(); err != nil {
		return models.TestReport{}, fmt.Errorf("cannot execute solution: %w", err)
	}
	report, err := process.Wait()
	if err != nil {
		return models.TestReport{}, fmt.Errorf("cannot wait solution: %w", err)
	}
	testReport := models.TestReport{
		Verdict: models.Accepted,
		Usage: models.UsageReport{
			Time:   report.Time.Milliseconds(),
			Memory: report.Memory,
		},
	}
	if report.Time.Milliseconds() > testSet.TimeLimit() {
		testReport.Verdict = models.TimeLimitExceeded
	} else if report.Memory > testSet.MemoryLimit() {
		testReport.Verdict = models.MemoryLimitExceeded
	} else if report.ExitCode != 0 {
		testReport.Verdict = models.RuntimeError
	}
	return testReport, nil
}

func (t *judgeSolutionTask) executeInteractiveSolution(
	ctx context.Context,
	testSet problems.ProblemTestSet,
	inputPath, outputPath, answerPath string,
) (models.TestReport, error) {
	interactorReader, interactorWriter, err := os.Pipe()
	if err != nil {
		return models.TestReport{}, err
	}
	defer func() {
		_ = interactorReader.Close()
		_ = interactorWriter.Close()
	}()
	solutionReader, solutionWriter, err := os.Pipe()
	if err != nil {
		return models.TestReport{}, err
	}
	defer func() {
		_ = solutionReader.Close()
		_ = solutionWriter.Close()
	}()
	interactorLog := truncateBuffer{limit: 2048}
	interactorProcess, err := t.interactorImpl.CreateProcess(ctx, compilers.ExecuteOptions{
		Args:        []string{"input.in", "output.out", "answer.ans"},
		Stdin:       solutionReader,
		Stdout:      interactorWriter,
		Stderr:      &interactorLog,
		TimeLimit:   2 * time.Duration(testSet.TimeLimit()) * time.Millisecond,
		MemoryLimit: 256 * 1024 * 1024,
	})
	if err != nil {
		return models.TestReport{}, fmt.Errorf("cannot prepare interactor: %w", err)
	}
	defer func() { _ = interactorProcess.Release() }()
	if err := copyFileRec(inputPath, interactorProcess.UpperPath("input.in")); err != nil {
		return models.TestReport{}, err
	}
	if err := copyFileRec(answerPath, interactorProcess.UpperPath("answer.ans")); err != nil {
		return models.TestReport{}, err
	}
	solutionProcess, err := t.solutionImpl.CreateProcess(ctx, compilers.ExecuteOptions{
		Stdin:       interactorReader,
		Stdout:      solutionWriter,
		TimeLimit:   time.Duration(testSet.TimeLimit()) * time.Millisecond,
		MemoryLimit: testSet.MemoryLimit(),
	})
	if err != nil {
		return models.TestReport{}, fmt.Errorf("cannot prepare solution: %w", err)
	}
	defer func() { _ = solutionProcess.Release() }()
	if err := interactorProcess.Start(); err != nil {
		return models.TestReport{}, fmt.Errorf("cannot execute interactor: %w", err)
	}
	if err := solutionProcess.Start(); err != nil {
		return models.TestReport{}, fmt.Errorf("cannot execute solution: %w", err)
	}
	interactorReportFuture := futures.Call(func() (safeexec.Report, error) {
		defer func() {
			_ = solutionReader.Close()
			_ = interactorWriter.Close()
		}()
		return interactorProcess.Wait()
	})
	solutionReportFuture := futures.Call(func() (safeexec.Report, error) {
		defer func() {
			_ = interactorReader.Close()
			_ = solutionWriter.Close()
		}()
		return solutionProcess.Wait()
	})
	interactorReport, err := interactorReportFuture.Get(ctx)
	if err != nil {
		return models.TestReport{}, fmt.Errorf("cannot wait interactor: %w", err)
	}
	solutionReport, err := solutionReportFuture.Get(ctx)
	if err != nil {
		return models.TestReport{}, fmt.Errorf("cannot wait solution: %w", err)
	}
	if err := copyFileRec(interactorProcess.UpperPath("output.out"), outputPath); err != nil {
		return models.TestReport{}, err
	}
	testReport := models.TestReport{
		Verdict: models.Accepted,
		Usage: models.UsageReport{
			Time:   solutionReport.Time.Milliseconds(),
			Memory: solutionReport.Memory,
		},
	}
	if solutionReport.Time.Milliseconds() > testSet.TimeLimit() {
		testReport.Verdict = models.TimeLimitExceeded
	} else if solutionReport.Memory > testSet.MemoryLimit() {
		testReport.Verdict = models.MemoryLimitExceeded
	} else if solutionReport.ExitCode != 0 {
		testReport.Verdict = models.RuntimeError
	} else {
		verdict, err := getTestlibExitCodeVerdict(interactorReport.ExitCode)
		if err != nil {
			return models.TestReport{}, err
		}
		testReport.Verdict = verdict
		testReport.Interactor = &models.ExecuteReport{
			Usage: models.UsageReport{
				Time:   interactorReport.Time.Milliseconds(),
				Memory: interactorReport.Memory,
			},
			Log: interactorLog.String(),
		}
	}
	return testReport, nil
}

func (t *judgeSolutionTask) runSolutionTest(
	ctx TaskContext,
	testSet problems.ProblemTestSet,
	test problems.ProblemTest,
) (models.TestReport, error) {
	inputPath := filepath.Join(t.tempDir, "test.in")
	outputPath := filepath.Join(t.tempDir, "test.out")
	answerPath := filepath.Join(t.tempDir, "test.ans")
	// Copy input.
	if err := func() error {
		testFile, err := test.OpenInput()
		if err != nil {
			return err
		}
		file, err := os.Create(inputPath)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		if _, err := io.Copy(file, testFile); err != nil {
			return err
		}
		return file.Sync()
	}(); err != nil {
		return models.TestReport{}, err
	}
	// Copy output.
	if err := func() error {
		testFile, err := test.OpenAnswer()
		if err != nil {
			return err
		}
		file, err := os.Create(answerPath)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		if _, err := io.Copy(file, testFile); err != nil {
			return err
		}
		return file.Sync()
	}(); err != nil {
		return models.TestReport{}, err
	}
	testReport, err := t.executeSolution(
		ctx, testSet, inputPath, outputPath, answerPath,
	)
	if err != nil {
		return models.TestReport{}, err
	}
	if testReport.Verdict != models.Accepted {
		return testReport, nil
	}
	checkerReport, err := runTestlibChecker(ctx, t.checkerImpl, inputPath, outputPath, answerPath)
	if err != nil {
		return models.TestReport{}, err
	}
	testReport.Verdict = checkerReport.Verdict
	testReport.Checker = checkerReport.Checker
	if testReport.Verdict == models.Accepted {
		if points := test.Points(); points > 0 {
			testReport.Points = &points
		}
	}
	return testReport, nil
}

func (t *judgeSolutionTask) runSolutionTests(
	ctx TaskContext, report *models.SolutionReport,
) error {
	state := models.JudgeSolutionTaskState{
		Stage: "testing",
	}
	if err := ctx.SetDeferredState(state); err != nil {
		return err
	}
	testSets, err := t.problemImpl.GetTestSets()
	if err != nil {
		return err
	}
	report.Verdict = models.Accepted
	if t.config.EnablePoints {
		points := float64(0)
		report.Points = &points
	}
	testNumber := 0
	for _, testSet := range testSets {
		tests, err := testSet.GetTests()
		if err != nil {
			return err
		}
		groupTests := map[string][]int{}
		for _, test := range tests {
			testNumber++
			state.Test = testNumber
			if err := ctx.SetDeferredState(state); err != nil {
				return err
			}
			testReport, err := t.runSolutionTest(ctx, testSet, test)
			if err != nil {
				return err
			}
			if !t.config.EnablePoints {
				testReport.Points = nil
			}
			groupTests[test.Group()] = append(
				groupTests[test.Group()], len(report.Tests),
			)
			report.Tests = append(report.Tests, testReport)
			if report.Usage.Time < testReport.Usage.Time {
				report.Usage.Time = testReport.Usage.Time
			}
			if report.Usage.Memory < testReport.Usage.Memory {
				report.Usage.Memory = testReport.Usage.Memory
			}
			ctx.Logger().Debug(
				"Solution test completed",
				logs.Any("test", testNumber),
				logs.Any("verdict", testReport.Verdict.String()),
			)
			if testReport.Verdict != models.Accepted {
				if t.config.EnablePoints {
					report.Verdict = models.PartiallyAccepted
				} else {
					// Stop judging immediately.
					report.Verdict = testReport.Verdict
					return nil
				}
			}
		}
		if t.config.EnablePoints {
			if err := t.calculateTestSetPoints(
				ctx, report, testSet, groupTests,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *judgeSolutionTask) executeImpl(ctx TaskContext, compileCtx problems.CompileContext) error {
	if err := t.prepareProblem(ctx); err != nil {
		return fmt.Errorf("cannot prepare problem: %w", err)
	}
	if err := t.prepareSolution(ctx); err != nil {
		return fmt.Errorf("cannot prepare solution: %w", err)
	}
	report := models.SolutionReport{
		Verdict: models.Rejected,
	}
	if ok, err := t.compileSolution(ctx, &report); err != nil {
		return fmt.Errorf("cannot compile solution: %w", err)
	} else if !ok {
		report.Verdict = models.CompilationError
	} else {
		if err := t.prepareExecutables(ctx, compileCtx); err != nil {
			return err
		}
		if err := t.runSolutionTests(ctx, &report); err != nil {
			return fmt.Errorf("cannot judge solution: %w", err)
		}
	}
	if err := t.solution.SetReport(&report); err != nil {
		return err
	}
	return t.invoker.core.Solutions.Update(ctx, t.solution)
}

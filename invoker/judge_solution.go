package invoker

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/udovin/solve/models"
	"github.com/udovin/solve/pkg/logs"
)

func init() {
	registerTaskImpl(models.JudgeSolutionTask, &judgeSolutionTask{})
}

type judgeSolutionTask struct {
	invoker      *Invoker
	config       models.JudgeSolutionTaskConfig
	solution     models.Solution
	problem      models.Problem
	compiler     models.Compiler
	tempDir      string
	problemImpl  Problem
	compilerImpl Compiler
	solutionPath string
	compiledPath string
	checker      *testlibCheckerImpl
	interactor   *testlibCheckerImpl
}

func (judgeSolutionTask) New(invoker *Invoker) taskImpl {
	return &judgeSolutionTask{invoker: invoker}
}

func (t *judgeSolutionTask) Execute(ctx TaskContext) error {
	// Fetch information about task.
	if err := ctx.ScanConfig(&t.config); err != nil {
		return fmt.Errorf("unable to scan task config: %w", err)
	}
	solution, err := t.invoker.getSolution(ctx, t.config.SolutionID)
	if err != nil {
		return fmt.Errorf("unable to fetch solution: %w", err)
	}
	problem, err := t.invoker.core.Problems.Get(ctx, solution.ProblemID)
	if err != nil {
		return fmt.Errorf("unable to fetch problem: %w", err)
	}
	compiler, err := t.invoker.core.Compilers.Get(ctx, solution.CompilerID)
	if err != nil {
		return fmt.Errorf("unable to fetch compiler: %w", err)
	}
	tempDir, err := makeTempDir()
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tempDir) }()
	t.tempDir = tempDir
	t.solution = solution
	t.problem = problem
	t.compiler = compiler
	return t.executeImpl(ctx)
}

func (t *judgeSolutionTask) prepareProblem(ctx TaskContext) error {
	if t.problem.PackageID == 0 {
		return fmt.Errorf("problem does not have package")
	}
	problem, err := t.invoker.problems.DownloadProblem(
		ctx, t.problem, CompiledProblem,
	)
	if err != nil {
		return fmt.Errorf("cannot download problem: %w", err)
	}
	t.problemImpl = problem
	return nil
}

func (t *judgeSolutionTask) prepareCompiler(ctx TaskContext) error {
	compiler, err := t.invoker.compilers.DownloadCompiler(ctx, t.compiler)
	if err != nil {
		return fmt.Errorf("cannot download compiler: %w", err)
	}
	t.compilerImpl = compiler
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
	compileReport, err := t.compilerImpl.Compile(ctx, CompileOptions{
		Source:      t.solutionPath,
		Target:      t.compiledPath,
		TimeLimit:   20 * time.Second,
		MemoryLimit: 256 * 1024 * 1024,
	})
	if err != nil {
		return false, err
	}
	report.Compile = models.CompileReport{
		Log: compileReport.Log,
		Usage: models.UsageReport{
			Time:   compileReport.UsedTime.Milliseconds(),
			Memory: compileReport.UsedMemory,
		},
	}
	return compileReport.Success(), nil
}

type testlibCheckerImpl struct {
	compiler   Compiler
	binaryPath string
	tempDir    string
}

func (c *testlibCheckerImpl) Run(
	ctx context.Context, inputPath, outputPath, answerPath string,
) (models.TestReport, error) {
	checkerLogPath := filepath.Join(c.tempDir, "checker.log")
	checkerReport, err := c.compiler.Execute(ctx, ExecuteOptions{
		Binary: c.binaryPath,
		Args:   []string{"input.in", "output.out", "answer.ans"},
		InputFiles: []MountFile{
			{Source: inputPath, Target: "input.in"},
			{Source: outputPath, Target: "output.out"},
			{Source: answerPath, Target: "answer.ans"},
		},
		OutputFiles: []MountFile{
			{Source: checkerLogPath, Target: "stderr"},
		},
		TimeLimit:   20 * time.Second,
		MemoryLimit: 256 * 1024 * 1024,
	})
	if err != nil {
		return models.TestReport{}, fmt.Errorf("cannot check solution: %w", err)
	}
	report := models.TestReport{}
	switch checkerReport.ExitCode {
	case 0:
		report.Verdict = models.Accepted
	case 1:
		report.Verdict = models.WrongAnswer
	case 3:
		report.Verdict = models.Failed
	case 2, 4, 8:
		report.Verdict = models.PresentationError
	case 5:
		report.Verdict = models.PartiallyAccepted
	default:
		if checkerReport.ExitCode < 16 {
			return models.TestReport{}, fmt.Errorf("checker exited with code: %d", checkerReport.ExitCode)
		}
		report.Verdict = models.PartiallyAccepted
	}
	checkerLog, err := readFile(checkerLogPath, 256)
	if err != nil {
		return models.TestReport{}, err
	}
	report.Check = models.CheckReport{
		Log: checkerLog,
		Usage: models.UsageReport{
			Time:   checkerReport.UsedTime.Milliseconds(),
			Memory: checkerReport.UsedMemory,
		},
	}
	return report, nil
}

var (
	errNoChecker    = fmt.Errorf("cannot find checker executable")
	errNoInteractor = fmt.Errorf("cannot find interactor executable")
)

func (t *judgeSolutionTask) getChecker(
	ctx TaskContext, executables []ProblemExecutable,
) (*testlibCheckerImpl, error) {
	var checker ProblemExecutable
	for _, executable := range executables {
		if executable.Kind() == TestlibChecker {
			checker = executable
			break
		}
	}
	if checker == nil {
		return nil, errNoChecker
	}
	compiler, err := t.invoker.compilers.GetCompiler(ctx, checker.Compiler())
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
	return &testlibCheckerImpl{
		compiler:   compiler,
		binaryPath: checkerPath,
		tempDir:    t.tempDir,
	}, nil
}

func (t *judgeSolutionTask) getInteractor(
	ctx TaskContext, executables []ProblemExecutable,
) (*testlibCheckerImpl, error) {
	var interactor ProblemExecutable
	for _, executable := range executables {
		if executable.Kind() == TestlibInteractor {
			interactor = executable
			break
		}
	}
	if interactor == nil {
		return nil, errNoInteractor
	}
	compiler, err := t.invoker.compilers.GetCompiler(ctx, interactor.Compiler())
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
	return &testlibCheckerImpl{
		compiler:   compiler,
		binaryPath: interactorPath,
		tempDir:    t.tempDir,
	}, nil
}

func (t *judgeSolutionTask) calculateTestSetPoints(
	ctx TaskContext,
	report *models.SolutionReport,
	testSet ProblemTestSet,
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
		case EachTestPointsPolicy:
			*report.Points += groupPoints
		case CompleteGroupPointsPolicy:
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

func (t *judgeSolutionTask) prepareExecutables(ctx TaskContext) error {
	executables, err := t.problemImpl.GetExecutables()
	if err != nil {
		return fmt.Errorf("cannot get executables: %w", err)
	}
	t.checker, err = t.getChecker(ctx, executables)
	if err != nil {
		return err
	}
	t.interactor, err = t.getInteractor(ctx, executables)
	if err != nil && err != errNoInteractor {
		return err
	}
	return nil
}

func (t *judgeSolutionTask) executeSolution(
	ctx context.Context, testSet ProblemTestSet, inputPath, outputPath string,
) (models.TestReport, error) {
	executeReport, err := t.compilerImpl.Execute(ctx, ExecuteOptions{
		Binary: t.compiledPath,
		InputFiles: []MountFile{
			{Source: inputPath, Target: "stdin"},
		},
		OutputFiles: []MountFile{
			{Source: outputPath, Target: "stdout"},
		},
		TimeLimit:   time.Duration(testSet.TimeLimit()) * time.Millisecond,
		MemoryLimit: testSet.MemoryLimit(),
	})
	if err != nil {
		return models.TestReport{}, fmt.Errorf("cannot execute solution: %w", err)
	}
	// Read solution input.
	input, err := readFile(inputPath, 128)
	if err != nil {
		return models.TestReport{}, err
	}
	// Read solution output.
	output, err := readFile(outputPath, 128)
	if err != nil {
		return models.TestReport{}, err
	}
	testReport := models.TestReport{
		Verdict: models.Accepted,
		Input:   input,
		Output:  output,
		Usage: models.UsageReport{
			Time:   executeReport.UsedTime.Milliseconds(),
			Memory: executeReport.UsedMemory,
		},
	}
	if executeReport.UsedTime.Milliseconds() > testSet.TimeLimit() {
		testReport.Verdict = models.TimeLimitExceeded
	} else if executeReport.UsedMemory > testSet.MemoryLimit() {
		testReport.Verdict = models.MemoryLimitExceeded
	} else if !executeReport.Success() {
		testReport.Verdict = models.RuntimeError
	}
	return testReport, nil
}

func (t *judgeSolutionTask) runSolutionTest(
	ctx TaskContext,
	testSet ProblemTestSet,
	test ProblemTest,
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
	testReport, err := t.executeSolution(ctx, testSet, inputPath, outputPath)
	if err != nil {
		return models.TestReport{}, err
	}
	if testReport.Verdict != models.Accepted {
		return testReport, nil
	}
	checkerReport, err := t.checker.Run(ctx, inputPath, outputPath, answerPath)
	if err != nil {
		return models.TestReport{}, err
	}
	testReport.Verdict = checkerReport.Verdict
	testReport.Check = checkerReport.Check
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

func (t *judgeSolutionTask) executeImpl(ctx TaskContext) error {
	if err := t.prepareProblem(ctx); err != nil {
		return fmt.Errorf("cannot prepare problem: %w", err)
	}
	if err := t.prepareCompiler(ctx); err != nil {
		return fmt.Errorf("cannot prepare compiler: %w", err)
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
		if err := t.prepareExecutables(ctx); err != nil {
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

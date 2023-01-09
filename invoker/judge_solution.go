package invoker

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/udovin/solve/models"
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
	problem, err := t.invoker.core.Problems.Get(solution.ProblemID)
	if err != nil {
		return fmt.Errorf("unable to fetch problem: %w", err)
	}
	compiler, err := t.invoker.core.Compilers.Get(solution.CompilerID)
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
	if err := ctx.SetState(ctx, state); err != nil {
		return false, err
	}
	compileReport, err := t.compilerImpl.Compile(ctx, CompileOptions{
		Source: t.solutionPath,
		Target: t.compiledPath,
	})
	if err != nil {
		return false, err
	}
	report.Compile = models.CompileReport{
		Log: compileReport.Log,
	}
	return compileReport.Success(), nil
}

func (t *judgeSolutionTask) testSolution(
	ctx TaskContext, report *models.SolutionReport,
) error {
	state := models.JudgeSolutionTaskState{
		Stage: "testing",
	}
	if err := ctx.SetState(ctx, state); err != nil {
		return err
	}
	executables, err := t.problemImpl.GetExecutables()
	if err != nil {
		return fmt.Errorf("cannot get executables: %w", err)
	}
	var checker ProblemExecutable
	for _, executable := range executables {
		if executable.Kind() == TestlibChecker {
			checker = executable
			break
		}
	}
	if checker == nil {
		return fmt.Errorf("cannot find checker executable")
	}
	checkerCompiler, err := t.invoker.compilers.GetCompiler(ctx, checker.Compiler())
	if err != nil {
		return err
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
		_, err = io.Copy(file, testFile)
		return err
	}(); err != nil {
		return err
	}
	groups, err := t.problemImpl.GetTestGroups()
	if err != nil {
		return err
	}
	for _, group := range groups {
		tests, err := group.GetTests()
		if err != nil {
			return err
		}
		for _, test := range tests {
			inputPath := filepath.Join(t.tempDir, "test.in")
			outputPath := filepath.Join(t.tempDir, "test.out")
			answerPath := filepath.Join(t.tempDir, "test.ans")
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
				_, err = io.Copy(file, testFile)
				return err
			}(); err != nil {
				return err
			}
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
				_, err = io.Copy(file, testFile)
				return err
			}(); err != nil {
				return err
			}
			executeReport, err := t.compilerImpl.Execute(ctx, ExecuteOptions{
				Binary: t.compiledPath,
				InputFiles: []MountFile{
					{Source: inputPath, Target: "stdin"},
				},
				OutputFiles: []MountFile{
					{Source: outputPath, Target: "stdout"},
				},
			})
			if err != nil {
				return fmt.Errorf("cannot execute solution: %w", err)
			}
			input, err := readFile(inputPath, 128)
			if err != nil {
				return err
			}
			output, err := readFile(outputPath, 128)
			if err != nil {
				return err
			}
			testReport := models.TestReport{
				Verdict: models.Rejected,
				Input:   input,
				Output:  output,
			}
			if !executeReport.Success() {
				testReport.Verdict = models.RuntimeError
			} else {
				checkerLogPath := filepath.Join(t.tempDir, "checker.log")
				checkerReport, err := checkerCompiler.Execute(ctx, ExecuteOptions{
					Binary: checkerPath,
					Args:   []string{"input.in", "output.out", "answer.ans"},
					InputFiles: []MountFile{
						{Source: inputPath, Target: "input.in"},
						{Source: outputPath, Target: "output.out"},
						{Source: answerPath, Target: "answer.ans"},
					},
					OutputFiles: []MountFile{
						{Source: checkerLogPath, Target: "stderr"},
					},
				})
				if err != nil {
					return err
				}
				switch checkerReport.ExitCode {
				case 0:
					testReport.Verdict = models.Accepted
				case 1:
					testReport.Verdict = models.WrongAnswer
				case 2, 4, 8:
					testReport.Verdict = models.PresentationError
				case 5:
					testReport.Verdict = models.PartiallyAccepted
				default:
					if checkerReport.ExitCode < 16 {
						return fmt.Errorf("checker exited with code: %d", checkerReport.ExitCode)
					}
					testReport.Verdict = models.PartiallyAccepted
				}
				checkerLog, err := readFile(checkerLogPath, 256)
				if err != nil {
					return err
				}
				testReport.Check.Log = checkerLog
			}
			report.Tests = append(report.Tests, testReport)
			if testReport.Verdict != models.Accepted {
				report.Verdict = testReport.Verdict
				return nil
			}
		}
	}
	report.Verdict = models.Accepted
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
		if err := t.testSolution(ctx, &report); err != nil {
			return fmt.Errorf("cannot judge solution: %w", err)
		}
	}
	if err := t.solution.SetReport(&report); err != nil {
		return err
	}
	return t.invoker.core.Solutions.Update(ctx, t.solution)
}

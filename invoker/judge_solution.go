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
	}
	if err := t.solution.SetReport(&report); err != nil {
		return err
	}
	return t.invoker.core.Solutions.Update(ctx, t.solution)
}

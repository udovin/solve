package invoker

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/udovin/solve/models"
	"github.com/udovin/solve/pkg"
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
	problemPath  string
	compilerPath string
	solutionPath string
}

func (judgeSolutionTask) New(invoker *Invoker) taskImpl {
	return &judgeSolutionTask{invoker: invoker}
}

func (t *judgeSolutionTask) Execute(ctx TaskContext) error {
	// Fetch information about task.
	if err := ctx.ScanConfig(&t.config); err != nil {
		return fmt.Errorf("unable to scan task config: %w", err)
	}
	solution, err := t.invoker.getSolution(t.config.SolutionID)
	if err != nil {
		return fmt.Errorf("unable to fetch task solution: %w", err)
	}
	problem, err := t.invoker.core.Problems.Get(solution.ProblemID)
	if err != nil {
		return fmt.Errorf("unable to fetch task problem: %w", err)
	}
	compiler, err := t.invoker.core.Compilers.Get(solution.CompilerID)
	if err != nil {
		return fmt.Errorf("unable to fetch task compiler: %w", err)
	}
	t.solution = solution
	t.problem = problem
	t.compiler = compiler
	return t.executeImpl(ctx)
}

func (t *judgeSolutionTask) prepareProblem(ctx TaskContext) error {
	problemFile, err := t.invoker.files.DownloadFile(ctx, t.problem.PackageID)
	if err != nil {
		return err
	}
	defer problemFile.Close()
	tempProblemPath := filepath.Join(t.tempDir, "problem")
	if err := pkg.ExtractZip(problemFile.Name(), tempProblemPath); err != nil {
		return err
	}
	t.problemPath = tempProblemPath
	return nil
}

func (t *judgeSolutionTask) prepareCompiler(ctx TaskContext) error {
	compilerFile, err := t.invoker.files.DownloadFile(ctx, t.compiler.ImageID)
	if err != nil {
		return err
	}
	defer compilerFile.Close()
	tempCompilerPath := filepath.Join(t.tempDir, "compiler")
	if err := pkg.ExtractTarGz(compilerFile.Name(), tempCompilerPath); err != nil {
		return err
	}
	t.compilerPath = tempCompilerPath
	return nil
}

func (t *judgeSolutionTask) prepareSolution(ctx TaskContext) error {
	if t.solution.ContentID == 0 {
		tempSolutionPath := filepath.Join(t.tempDir, "solution.txt")
		err := ioutil.WriteFile(tempSolutionPath, []byte(t.solution.Content), fs.ModePerm)
		if err != nil {
			return err
		}
		t.solutionPath = tempSolutionPath
		return nil
	}
	solutionFile, err := t.invoker.files.DownloadFile(ctx, int64(t.solution.ContentID))
	if err != nil {
		return err
	}
	defer solutionFile.Close()
	tempSolutionPath := filepath.Join(t.tempDir, "solution.bin")
	file, err := os.Create(tempSolutionPath)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(file, solutionFile); err != nil {
		return err
	}
	t.solutionPath = tempSolutionPath
	return nil
}

func (t *judgeSolutionTask) executeImpl(ctx TaskContext) error {
	tempDir, err := makeTempDir()
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()
	t.tempDir = tempDir
	if err := t.prepareProblem(ctx); err != nil {
		return err
	}
	if err := t.prepareCompiler(ctx); err != nil {
		return err
	}
	if err := t.prepareSolution(ctx); err != nil {
		return err
	}
	return fmt.Errorf("not implemented")
}

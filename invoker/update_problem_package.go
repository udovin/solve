package invoker

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/udovin/solve/models"
	"github.com/udovin/solve/pkg"
	"github.com/udovin/solve/pkg/polygon"
)

func init() {
	registerTaskImpl(models.UpdateProblemPackageTask, &updateProblemPackageTask{})
}

type updateProblemPackageTask struct {
	invoker     *Invoker
	config      models.UpdateProblemPackageTaskConfig
	problem     models.Problem
	file        models.File
	resources   []models.ProblemResource
	tempDir     string
	problemPath string
}

func (updateProblemPackageTask) New(invoker *Invoker) taskImpl {
	return &judgeSolutionTask{invoker: invoker}
}

func (t *updateProblemPackageTask) Execute(ctx TaskContext) error {
	if err := ctx.ScanConfig(&t.config); err != nil {
		return fmt.Errorf("unable to scan task config: %w", err)
	}
	problem, err := t.invoker.core.Problems.Get(t.config.ProblemID)
	if err != nil {
		return fmt.Errorf("unable to fetch problem: %w", err)
	}
	file, err := t.invoker.core.Files.Get(t.config.FileID)
	if err != nil {
		return fmt.Errorf("unable to fetch problem: %w", err)
	}
	resources, err := t.invoker.core.ProblemResources.FindByProblem(
		problem.ID,
	)
	if err != nil {
		return fmt.Errorf("unable to fetch resources: %w", err)
	}
	tempDir, err := makeTempDir()
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tempDir) }()
	t.tempDir = tempDir
	t.problem = problem
	t.file = file
	t.resources = resources
	return t.executeImpl(ctx)
}

func (t *updateProblemPackageTask) prepareProblem(ctx TaskContext) error {
	if t.problem.PackageID == 0 {
		return fmt.Errorf("problem does not have package")
	}
	problemFile, err := t.invoker.files.DownloadFile(ctx, int64(t.problem.PackageID))
	if err != nil {
		return fmt.Errorf("cannot download problem: %w", err)
	}
	defer func() { _ = problemFile.Close() }()
	tempProblemPath := filepath.Join(t.tempDir, "problem")
	if err := pkg.ExtractZip(problemFile.Name(), tempProblemPath); err != nil {
		return fmt.Errorf("cannot extract problem: %w", err)
	}
	t.problemPath = tempProblemPath
	return nil
}

func (t *updateProblemPackageTask) executeImpl(ctx TaskContext) error {
	if err := t.prepareProblem(ctx); err != nil {
		return fmt.Errorf("cannot prepare problem: %w", err)
	}
	problem, err := polygon.ReadProblem(t.problemPath)
	if err != nil {
		return fmt.Errorf("cannot read problem: %w", err)
	}
	resources := []models.ProblemResource{}
	for _, statement := range problem.Statements {
		locale, ok := polygonLocales[statement.Language]
		if !ok {
			continue
		}
		if statement.Type != "application/x-tex" {
			continue
		}
		properties, err := polygon.ReadProblemProperites(
			t.problemPath, statement.Language,
		)
		if err != nil {
			return err
		}
		config := models.ProblemStatementConfig{
			Locale: locale,
			Title:  properties.Name,
			Legend: properties.Legend,
			Input:  properties.Input,
			Output: properties.Output,
			Notes:  properties.Notes,
		}
		resource := models.ProblemResource{}
		if err := resource.SetConfig(config); err != nil {
			return err
		}
		resources = append(resources, resource)
	}
	return nil
}

var polygonLocales = map[string]string{
	"russian": "ru",
	"english": "en",
}

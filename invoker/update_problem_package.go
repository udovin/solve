package invoker

import (
	"context"
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
	return &updateProblemPackageTask{invoker: invoker}
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
	if t.file.ID == 0 {
		return fmt.Errorf("problem does not have package")
	}
	problemFile, err := t.invoker.files.DownloadFile(ctx, int64(t.file.ID))
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
	events := map[string]models.ProblemResourceEvent{}
	for _, resource := range t.resources {
		if resource.Kind != models.ProblemStatement {
			continue
		}
		config := models.ProblemStatementConfig{}
		if err := resource.ScanConfig(&config); err != nil {
			continue
		}
		event := models.ProblemResourceEvent{ProblemResource: resource}
		event.BaseEventKind = models.DeleteEvent
		events[config.Locale] = event
	}
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
		event, ok := events[locale]
		if !ok {
			event.BaseEventKind = models.CreateEvent
			event.ProblemID = t.problem.ID
		} else {
			event.BaseEventKind = models.UpdateEvent
		}
		if err := event.ProblemResource.SetConfig(config); err != nil {
			return err
		}
		events[locale] = event
	}
	return t.invoker.core.WrapTx(ctx, func(ctx context.Context) error {
		for _, event := range events {
			switch event.BaseEventKind {
			case models.CreateEvent:
				if err := t.invoker.core.ProblemResources.Create(
					ctx, &event.ProblemResource,
				); err != nil {
					return err
				}
			case models.UpdateEvent:
				if err := t.invoker.core.ProblemResources.Update(
					ctx, event.ProblemResource,
				); err != nil {
					return err
				}
			case models.DeleteEvent:
				if err := t.invoker.core.ProblemResources.Delete(
					ctx, event.ProblemResource.ID,
				); err != nil {
					return err
				}
			default:
				return fmt.Errorf(
					"unsupported kind: %v", event.BaseEventKind,
				)
			}
		}
		t.problem.PackageID = models.NInt64(t.file.ID)
		return t.invoker.core.Problems.Update(ctx, t.problem)
	}, sqlRepeatableRead)
}

var polygonLocales = map[string]string{
	"russian": "ru",
	"english": "en",
}

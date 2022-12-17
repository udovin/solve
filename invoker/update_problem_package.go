package invoker

import (
	"context"
	"fmt"

	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

func init() {
	registerTaskImpl(models.UpdateProblemPackageTask, &updateProblemPackageTask{})
}

type updateProblemPackageTask struct {
	invoker        *Invoker
	config         models.UpdateProblemPackageTaskConfig
	problem        models.Problem
	file           models.File
	resources      []models.ProblemResource
	problemPackage Problem
}

func (updateProblemPackageTask) New(invoker *Invoker) taskImpl {
	return &updateProblemPackageTask{invoker: invoker}
}

func (t *updateProblemPackageTask) Execute(ctx TaskContext) error {
	if err := ctx.ScanConfig(&t.config); err != nil {
		return fmt.Errorf("unable to scan task config: %w", err)
	}
	if err := t.invoker.core.Problems.Sync(ctx); err != nil {
		return fmt.Errorf("unable to sync problems: %w", err)
	}
	problem, err := t.invoker.core.Problems.Get(t.config.ProblemID)
	if err != nil {
		return fmt.Errorf("unable to fetch problem: %w", err)
	}
	if err := t.invoker.core.Files.Sync(ctx); err != nil {
		return fmt.Errorf("unable to sync files: %w", err)
	}
	file, err := t.invoker.core.Files.Get(t.config.FileID)
	if err != nil {
		return fmt.Errorf("unable to fetch problem: %w", err)
	}
	if err := t.invoker.core.ProblemResources.Sync(ctx); err != nil {
		return fmt.Errorf("unable to sync resources: %w", err)
	}
	resources, err := t.invoker.core.ProblemResources.FindByProblem(
		problem.ID,
	)
	if err != nil {
		return fmt.Errorf("unable to fetch resources: %w", err)
	}
	t.problem = problem
	t.file = file
	t.resources = resources
	return t.executeImpl(ctx)
}

func (t *updateProblemPackageTask) prepareProblem(ctx TaskContext) error {
	if t.file.ID == 0 {
		return fmt.Errorf("problem does not have package")
	}
	problem, err := t.invoker.problems.DownloadProblem(ctx, t.file.ID)
	if err != nil {
		return fmt.Errorf("cannot download problem: %w", err)
	}
	t.problemPackage = problem
	return nil
}

func (t *updateProblemPackageTask) executeImpl(ctx TaskContext) error {
	if err := t.prepareProblem(ctx); err != nil {
		return fmt.Errorf("cannot prepare problem: %w", err)
	}
	type eventKey struct {
		Locale string
		Kind   models.ProblemResourceKind
		Name   string
	}
	events := map[eventKey]models.ProblemResourceEvent{}
	fileReaders := map[eventKey]*managers.FileReader{}
	defer func() {
		for _, file := range fileReaders {
			_ = file.Close()
		}
	}()
	for _, resource := range t.resources {
		key := eventKey{Kind: resource.Kind}
		switch resource.Kind {
		case models.ProblemStatement:
			config := models.ProblemStatementConfig{}
			if err := resource.ScanConfig(&config); err != nil {
				continue
			}
			key.Locale = config.Locale
			event := models.ProblemResourceEvent{ProblemResource: resource}
			event.BaseEventKind = models.DeleteEvent
			events[key] = event
		case models.ProblemStatementResource:
			config := models.ProblemStatementResourceConfig{}
			if err := resource.ScanConfig(&config); err != nil {
				continue
			}
			key.Locale = config.Locale
			key.Name = config.Name
			event := models.ProblemResourceEvent{ProblemResource: resource}
			event.BaseEventKind = models.DeleteEvent
			events[key] = event
		}
	}
	statements, err := t.problemPackage.GetStatements()
	if err != nil {
		return fmt.Errorf("cannot read problem: %w", err)
	}
	for _, statement := range statements {
		locale := statement.Locale()
		config, err := statement.GetConfig()
		if err != nil {
			return fmt.Errorf("cannot read statement: %w", err)
		}
		key := eventKey{
			Kind:   models.ProblemStatement,
			Locale: locale,
		}
		event, ok := events[key]
		if !ok {
			event.BaseEventKind = models.CreateEvent
			event.ProblemID = t.problem.ID
		} else {
			event.BaseEventKind = models.UpdateEvent
		}
		if err := event.ProblemResource.SetConfig(config); err != nil {
			return err
		}
		events[key] = event
		resources, err := statement.GetResources()
		if err != nil {
			return err
		}
		for _, resource := range resources {
			key := eventKey{
				Kind:   models.ProblemStatementResource,
				Locale: locale,
				Name:   resource.Name(),
			}
			config := models.ProblemStatementResourceConfig{
				Locale: locale,
				Name:   resource.Name(),
			}
			event, ok := events[key]
			if !ok {
				file, err := resource.Open()
				if err != nil {
					return err
				}
				fileReaders[key] = &managers.FileReader{
					Reader: file,
					Name:   resource.Name(),
				}
				event.BaseEventKind = models.CreateEvent
				event.ProblemID = t.problem.ID
			} else {
				if event.FileID != 0 {
					prevFile, err := t.invoker.core.Files.Get(int64(event.FileID))
					if err != nil {
						return err
					}
					fileMeta, err := prevFile.GetMeta()
					if err != nil {
						return err
					}
					md5, err := resource.GetMD5()
					if err != nil {
						return err
					}
					if fileMeta.MD5 == md5 {
						delete(events, key)
						continue
					}
				}
				file, err := resource.Open()
				if err != nil {
					return err
				}
				fileReaders[key] = &managers.FileReader{
					Reader: file,
					Name:   resource.Name(),
				}
				event.BaseEventKind = models.UpdateEvent
			}
			if err := event.ProblemResource.SetConfig(config); err != nil {
				return err
			}
			events[key] = event
		}
	}
	var files []models.File
	for key, fileReader := range fileReaders {
		event, ok := events[key]
		if !ok {
			continue
		}
		file, err := t.invoker.files.UploadFile(ctx, fileReader)
		if err != nil {
			return err
		}
		files = append(files, file)
		event.FileID = models.NInt64(file.ID)
		events[key] = event
	}
	return t.invoker.core.WrapTx(ctx, func(ctx context.Context) error {
		for _, file := range files {
			if err := t.invoker.files.ConfirmUploadFile(ctx, &file); err != nil {
				return err
			}
		}
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

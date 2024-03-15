package invoker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/udovin/solve/internal/db"
	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/problems"
	"github.com/udovin/solve/internal/pkg/problems/cache"
	"golang.org/x/exp/constraints"
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
	problemImpl problems.Problem
}

func (updateProblemPackageTask) New(invoker *Invoker) taskImpl {
	return &updateProblemPackageTask{invoker: invoker}
}

func (t *updateProblemPackageTask) Execute(ctx TaskContext) error {
	if err := ctx.ScanConfig(&t.config); err != nil {
		return fmt.Errorf("unable to scan task config: %w", err)
	}
	syncCtx := models.WithSync(ctx)
	problem, err := t.invoker.core.Problems.Get(syncCtx, t.config.ProblemID)
	if err != nil {
		return fmt.Errorf("unable to fetch problem: %w", err)
	}
	file, err := t.invoker.core.Files.Get(syncCtx, t.config.FileID)
	if err != nil {
		return fmt.Errorf("unable to fetch problem: %w", err)
	}
	resourceRows, err := t.invoker.core.ProblemResources.FindByProblem(syncCtx, problem.ID)
	if err != nil {
		return fmt.Errorf("unable to fetch resources: %w", err)
	}
	resources, err := db.CollectRows(resourceRows)
	if err != nil {
		return fmt.Errorf("unable to fetch resources: %w", err)
	}
	problemPackage, err := t.invoker.problemPackages.LoadSync(ctx, int64(problem.PackageID), problems.PolygonProblem)
	if err != nil {
		return fmt.Errorf("unable to fetch package: %w", err)
	}
	defer problemPackage.Release()
	tempDir, err := makeTempDir()
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tempDir) }()
	t.problem = problem
	t.problemImpl = problemPackage.Get()
	t.file = file
	t.resources = resources
	t.tempDir = tempDir
	if err := t.executeImpl(ctx); err != nil {
		state := models.UpdateProblemPackageTaskState{
			Error: err.Error(),
		}
		if err := ctx.SetDeferredState(&state); err != nil {
			ctx.Logger().Error("Cannot set deferred state", err)
		}
		return err
	}
	return nil
}

func max[T constraints.Ordered](a, b T) T {
	if a < b {
		return b
	}
	return a
}

func (t *updateProblemPackageTask) newCompileContext(ctx TaskContext) CompileContext {
	baseCtx := compileContext{
		compilers: t.invoker.core.Compilers,
		cache:     t.invoker.compilerImages,
		logger:    ctx.Logger(),
	}
	return &polygonCompileContext{ctx: &baseCtx, settings: t.invoker.core.Settings}
}

func (t *updateProblemPackageTask) compileProblem(ctx TaskContext, problemPath string) error {
	compileCtx := t.newCompileContext(ctx)
	defer compileCtx.Release()
	if err := t.problemImpl.Compile(ctx, compileCtx); err != nil {
		return fmt.Errorf("cannot compile problem: %w", err)
	}
	if err := cache.BuildCompiledProblem(
		ctx, compileCtx, t.problemImpl, problemPath,
	); err != nil {
		return fmt.Errorf("cannot build compiled problem: %w", err)
	}
	return nil
}

func (t *updateProblemPackageTask) executeImpl(ctx TaskContext) error {
	problemPath := filepath.Join(t.tempDir, "problem.zip")
	if t.config.Compile {
		if err := t.compileProblem(ctx, problemPath); err != nil {
			return err
		}
	}
	testSets, err := t.problemImpl.GetTestSets()
	if err != nil {
		return fmt.Errorf("cannot get test groups: %w", err)
	}
	config := models.ProblemConfig{}
	for _, testSet := range testSets {
		config.TimeLimit = max(config.TimeLimit, testSet.TimeLimit())
		config.MemoryLimit = max(config.MemoryLimit, testSet.MemoryLimit())
	}
	if err := t.problem.SetConfig(config); err != nil {
		return err
	}
	type eventKey struct {
		Locale string
		Kind   models.ProblemResourceKind
		Name   string
	}
	duplicates := []models.ProblemResource{}
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
			if duplicate, ok := events[key]; ok {
				duplicates = append(duplicates, duplicate.ProblemResource)
			}
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
			if duplicate, ok := events[key]; ok {
				duplicates = append(duplicates, duplicate.ProblemResource)
			}
			events[key] = event
		}
	}
	statements, err := t.problemImpl.GetStatements()
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
					prevFile, err := t.invoker.core.Files.Get(ctx, int64(event.FileID))
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
	if t.config.Compile {
		if file, err := os.Open(problemPath); err != nil {
			return fmt.Errorf("cannot open problem compiled package: %w", err)
		} else {
			defer func() { _ = file.Close() }()
			fileReader := &managers.FileReader{
				Reader: file,
				Name:   "problem.zip",
			}
			file, err := t.invoker.files.UploadFile(ctx, fileReader)
			if err != nil {
				return err
			}
			files = append(files, file)
			t.problem.CompiledID = models.NInt64(file.ID)
		}
	}
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
		for _, duplicate := range duplicates {
			if err := t.invoker.core.ProblemResources.Delete(
				ctx, duplicate.ID,
			); err != nil {
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
		return t.invoker.core.Problems.Update(ctx, t.problem)
	}, sqlRepeatableRead)
}

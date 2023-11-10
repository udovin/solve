package invoker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/udovin/algo/futures"
	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/archives"
	"github.com/udovin/solve/internal/pkg/hash"
	"github.com/udovin/solve/internal/pkg/logs"
	"github.com/udovin/solve/internal/pkg/polygon"
	"github.com/udovin/solve/internal/pkg/safeexec"
)

func extractPolygonProblem(source, target string) (Problem, error) {
	if err := archives.ExtractZip(source, target); err != nil {
		return nil, fmt.Errorf("cannot extract problem: %w", err)
	}
	config, err := polygon.ReadProblemConfig(
		filepath.Join(target, "problem.xml"),
	)
	if err != nil {
		_ = os.RemoveAll(target)
		return nil, fmt.Errorf("cannot read problem config: %w", err)
	}
	return &polygonProblem{
		path:   target,
		config: config,
	}, nil
}

type polygonProblem struct {
	path   string
	config polygon.Problem
}

func (p *polygonProblem) compileExecutable(
	ctx context.Context, compilers CompilerManager, executables map[string]Executable, polygonType string, source string, resources []MountFile,
) (Executable, error) {
	polygonName := "polygon." + polygonType
	compilerName, err := compilers.GetCompilerName(polygonName)
	if err != nil {
		return nil, err
	}
	compiler, err := compilers.GetCompiler(ctx, compilerName)
	if err != nil {
		return nil, err
	}
	target := strings.TrimSuffix(source, filepath.Ext(source))
	if _, ok := executables[target]; !ok {
		sourcePath := filepath.Join(p.path, source)
		targetPath := filepath.Join(p.path, target)
		report, err := compiler.Compile(ctx, CompileOptions{
			Source:      sourcePath,
			Target:      targetPath,
			InputFiles:  resources,
			TimeLimit:   20 * time.Second,
			MemoryLimit: 512 * 1024 * 1024,
		})
		if err != nil {
			return nil, err
		}
		if !report.Success() {
			return nil, fmt.Errorf(
				"cannot compile %q with compiler %q: %q",
				source, compilerName, report.Log,
			)
		}
		compilers.Logger().Debug(
			"Compiled executable",
			logs.Any("path", source),
		)
		executable, err := compiler.CreateExecutable(targetPath)
		if err != nil {
			return nil, err
		}
		executables[target] = executable
	}
	return executables[target], nil
}

func (p *polygonProblem) generateTest(ctx context.Context, executables map[string]Executable, args []string, input string) error {
	if len(args) == 0 {
		return fmt.Errorf("cannot find executable")
	}
	executable, ok := executables[fmt.Sprintf("files/%s", args[0])]
	if !ok {
		return fmt.Errorf("cannot find executable: %q", args[0])
	}
	outputFile, err := os.Create(filepath.Join(p.path, input))
	if err != nil {
		return fmt.Errorf("cannot find executable: %q", args[0])
	}
	defer func() { _ = outputFile.Close() }()
	process, err := executable.CreateProcess(ctx, ExecuteOptions{
		Args:        args[1:],
		Stdout:      outputFile,
		TimeLimit:   20 * time.Second,
		MemoryLimit: 256 * 1024 * 1024,
	})
	if err != nil {
		return fmt.Errorf("cannot create executable %q: %w", args[0], err)
	}
	defer func() { _ = process.Release() }()
	if err := process.Start(); err != nil {
		return fmt.Errorf("cannot start executable %q: %w", args[0], err)
	}
	report, err := process.Wait()
	if err != nil {
		return fmt.Errorf("cannot wait executable %q: %w", args[0], err)
	}
	if report.ExitCode != 0 {
		return fmt.Errorf("generator exited with code: %v", report.ExitCode)
	}
	return nil
}

func (p *polygonProblem) Compile(ctx context.Context, compilers CompilerManager) error {
	executables := map[string]Executable{}
	defer func() {
		for _, executable := range executables {
			_ = executable.Release()
		}
	}()
	resources := []MountFile{}
	for _, resource := range p.config.Files.Resources {
		if resource.Type != "h.g++" {
			continue
		}
		resources = append(resources, MountFile{
			Source: filepath.Join(p.path, resource.Path),
			Target: filepath.Base(resource.Path),
		})
	}
	for _, e := range p.config.Files.Executables {
		if e.Source == nil {
			continue
		}
		if _, err := p.compileExecutable(
			ctx, compilers, executables, e.Source.Type, e.Source.Path, resources,
		); err != nil {
			return err
		}
	}
	var interactor Executable
	if p.config.Assets != nil {
		if e := p.config.Assets.Checker; e != nil {
			if _, err := p.compileExecutable(
				ctx, compilers, executables, e.Source.Type, e.Source.Path, resources,
			); err != nil {
				return err
			}
		}
		if e := p.config.Assets.Interactor; e != nil {
			var err error
			interactor, err = p.compileExecutable(
				ctx, compilers, executables, e.Source.Type, e.Source.Path, resources,
			)
			if err != nil {
				return err
			}
		}
	}
	var mainSolution polygon.Solution
	for _, solution := range p.config.Assets.Solutions {
		if solution.Tag == "main" {
			mainSolution = solution
		}
	}
	if mainSolution.Source == nil {
		return fmt.Errorf("cannot find main solution")
	}
	var solution Executable
	{
		polygonName := "polygon." + mainSolution.Source.Type
		compilerName, err := compilers.GetCompilerName(polygonName)
		if err != nil {
			return err
		}
		compiler, err := compilers.GetCompiler(ctx, compilerName)
		if err != nil {
			return err
		}
		sourcePath := filepath.Join(p.path, mainSolution.Source.Path)
		targetPath := strings.TrimSuffix(sourcePath, filepath.Ext(sourcePath))
		report, err := compiler.Compile(ctx, CompileOptions{
			Source:      sourcePath,
			Target:      targetPath,
			TimeLimit:   20 * time.Second,
			MemoryLimit: 256 * 1024 * 1024,
		})
		if err != nil {
			return err
		}
		if !report.Success() {
			return fmt.Errorf(
				"cannot compile %q with compiler %q: %q",
				mainSolution.Source.Path, compilerName, report.Log,
			)
		}
		compilers.Logger().Debug(
			"Compiled solution",
			logs.Any("path", mainSolution.Source.Path),
		)
		solution, err = compiler.CreateExecutable(targetPath)
		if err != nil {
			return err
		}
		defer func() { _ = solution.Release() }()
	}
	for _, testSet := range p.config.TestSets {
		for i, test := range testSet.Tests {
			input := fmt.Sprintf(testSet.InputPathPattern, i+1)
			answer := fmt.Sprintf(testSet.AnswerPathPattern, i+1)
			if test.Cmd != "" {
				if err := p.generateTest(ctx, executables, strings.Fields(test.Cmd), input); err != nil {
					return err
				}
			}
			if interactor != nil {
				if err := func() error {
					interactorReader, interactorWriter, err := os.Pipe()
					if err != nil {
						return err
					}
					defer func() {
						_ = interactorReader.Close()
						_ = interactorWriter.Close()
					}()
					solutionReader, solutionWriter, err := os.Pipe()
					if err != nil {
						return err
					}
					defer func() {
						_ = solutionReader.Close()
						_ = solutionWriter.Close()
					}()
					process, err := interactor.CreateProcess(ctx, ExecuteOptions{
						Args:        []string{"input.in", "output.out"},
						Stdin:       solutionReader,
						Stdout:      interactorWriter,
						TimeLimit:   2 * time.Duration(testSet.TimeLimit) * time.Millisecond,
						MemoryLimit: 256 * 1024 * 1024,
					})
					if err != nil {
						return fmt.Errorf("cannot prepare interactor: %w", err)
					}
					defer func() { _ = process.Release() }()
					inputPath := filepath.Join(p.path, input)
					if err := copyFileRec(inputPath, process.UpperPath("input.in")); err != nil {
						return err
					}
					solutionProcess, err := solution.CreateProcess(ctx, ExecuteOptions{
						Stdin:       interactorReader,
						Stdout:      solutionWriter,
						TimeLimit:   time.Duration(testSet.TimeLimit) * time.Millisecond,
						MemoryLimit: testSet.MemoryLimit,
					})
					if err != nil {
						return fmt.Errorf("cannot prepare solution: %w", err)
					}
					defer func() { _ = solutionProcess.Release() }()
					if err := process.Start(); err != nil {
						return fmt.Errorf("cannot execute interactor: %w", err)
					}
					if err := solutionProcess.Start(); err != nil {
						return fmt.Errorf("cannot execute solution: %w", err)
					}
					interactorReportFuture := futures.Call(func() (safeexec.Report, error) {
						defer func() {
							_ = solutionReader.Close()
							_ = interactorWriter.Close()
						}()
						return process.Wait()
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
						return fmt.Errorf("cannot wait interactor: %w", err)
					}
					solutionReport, err := solutionReportFuture.Get(ctx)
					if err != nil {
						return fmt.Errorf("cannot wait solution: %w", err)
					}
					outputPath := filepath.Join(p.path, answer)
					if err := copyFileRec(process.UpperPath("output.out"), outputPath); err != nil {
						return err
					}
					if solutionReport.ExitCode != 0 {
						return fmt.Errorf("solution exited with code: %v", solutionReport.ExitCode)
					}
					verdict, err := getTestlibExitCodeVerdict(interactorReport.ExitCode)
					if err != nil {
						return err
					}
					if verdict != models.Accepted {
						return fmt.Errorf("interactor exited with verdict: %s", verdict)
					}
					return nil
				}(); err != nil {
					return err
				}
			} else {
				input, err := os.Open(filepath.Join(p.path, input))
				if err != nil {
					return fmt.Errorf("cannot open input file: %w", err)
				}
				defer func() { _ = input.Close() }()
				output, err := os.Create(filepath.Join(p.path, answer))
				if err != nil {
					return fmt.Errorf("cannot create output file: %w", err)
				}
				defer func() { _ = output.Close() }()
				process, err := solution.CreateProcess(ctx, ExecuteOptions{
					Stdin:       input,
					Stdout:      output,
					TimeLimit:   time.Duration(testSet.TimeLimit) * time.Millisecond,
					MemoryLimit: testSet.MemoryLimit,
				})
				if err != nil {
					return fmt.Errorf("cannot prepare solution: %w", err)
				}
				defer func() { _ = process.Release() }()
				if err := process.Start(); err != nil {
					return fmt.Errorf("cannot execute solution: %w", err)
				}
				report, err := process.Wait()
				if err != nil {
					return fmt.Errorf("cannot wait solution: %w", err)
				}
				if report.ExitCode != 0 {
					return fmt.Errorf("solution exited with code: %v", report.ExitCode)
				}
				if err := output.Sync(); err != nil {
					return fmt.Errorf("cannot sync output file: %w", err)
				}
			}
			compilers.Logger().Debug(
				"Generated test",
				logs.Any("input", input),
				logs.Any("answer", answer),
			)
		}
	}
	return nil
}

func (p *polygonProblem) GetExecutables() ([]ProblemExecutable, error) {
	var executables []ProblemExecutable
	if p.config.Assets == nil {
		return executables, nil
	}
	if p.config.Assets.Checker != nil {
		checker := p.config.Assets.Checker
		source := checker.Source.Path
		target := strings.TrimSuffix(source, filepath.Ext(source))
		targetPath := filepath.Join(p.path, target)
		executables = append(executables, polygonProblemExecutable{
			name:       "checker",
			kind:       TestlibChecker,
			binaryPath: targetPath,
			compiler:   checker.Source.Type,
		})
	}
	if p.config.Assets.Interactor != nil {
		interactor := p.config.Assets.Interactor
		source := interactor.Source.Path
		target := strings.TrimSuffix(source, filepath.Ext(source))
		targetPath := filepath.Join(p.path, target)
		executables = append(executables, polygonProblemExecutable{
			name:       "interactor",
			kind:       TestlibInteractor,
			binaryPath: targetPath,
			compiler:   interactor.Source.Type,
		})
	}
	return executables, nil
}

type polygonProblemExecutable struct {
	name       string
	kind       ProblemExecutableKind
	binaryPath string
	compiler   string
}

func (e polygonProblemExecutable) Name() string {
	return e.name
}

func (e polygonProblemExecutable) Kind() ProblemExecutableKind {
	return e.kind
}

func (e polygonProblemExecutable) OpenBinary() (*os.File, error) {
	return os.Open(e.binaryPath)
}

func (e polygonProblemExecutable) GetCompiler(ctx context.Context, compilers CompilerManager) (Compiler, error) {
	name, err := compilers.GetCompilerName("polygon." + e.compiler)
	if err != nil {
		return nil, err
	}
	return compilers.GetCompiler(ctx, name)
}

func (p *polygonProblem) GetTestSets() ([]ProblemTestSet, error) {
	var testSets []ProblemTestSet
	for _, testSet := range p.config.TestSets {
		testSets = append(testSets, &polygonProblemTestSet{
			problem: p,
			config:  testSet,
		})
	}
	return testSets, nil
}

func (p *polygonProblem) GetStatements() ([]ProblemStatement, error) {
	var statements []ProblemStatement
	for _, statement := range p.config.Statements {
		if statement.Type != "application/x-tex" {
			continue
		}
		if _, ok := polygonLocales[statement.Language]; !ok {
			continue
		}
		statements = append(statements, &polygonProblemStatement{
			problem:  p,
			language: statement.Language,
		})
	}
	return statements, nil
}

type polygonProblemTestSet struct {
	problem *polygonProblem
	config  polygon.TestSet
}

func (g *polygonProblemTestSet) Name() string {
	return g.config.Name
}

func (g *polygonProblemTestSet) TimeLimit() int64 {
	return g.config.TimeLimit
}

func (g *polygonProblemTestSet) MemoryLimit() int64 {
	return g.config.MemoryLimit
}

func (g *polygonProblemTestSet) GetGroups() ([]ProblemTestGroup, error) {
	var groups []ProblemTestGroup
	for _, group := range g.config.Groups {
		groups = append(groups, problemTestGroup{
			name:         group.Name,
			pointsPolicy: getPolygonPointsPolicy(group.PointsPolicy),
		})
	}
	return groups, nil
}

func getPolygonPointsPolicy(policy string) ProblemPointsPolicy {
	switch policy {
	case "each-test":
		return EachTestPointsPolicy
	case "complete-group":
		return CompleteGroupPointsPolicy
	default:
		return EachTestPointsPolicy
	}
}

func (g *polygonProblemTestSet) GetTests() ([]ProblemTest, error) {
	var tests []ProblemTest
	for i := range g.config.Tests {
		input := fmt.Sprintf(g.config.InputPathPattern, i+1)
		answer := fmt.Sprintf(g.config.AnswerPathPattern, i+1)
		tests = append(tests, problemTest{
			inputPath:  filepath.Join(g.problem.path, input),
			answerPath: filepath.Join(g.problem.path, answer),
			points:     g.config.Tests[i].Points,
			group:      g.config.Tests[i].Group,
		})
	}
	return tests, nil
}

type problemTestGroup struct {
	name         string
	pointsPolicy ProblemPointsPolicy
}

func (g problemTestGroup) Name() string {
	return g.name
}

func (g problemTestGroup) PointsPolicy() ProblemPointsPolicy {
	return g.pointsPolicy
}

type problemTest struct {
	inputPath  string
	answerPath string
	points     float64
	group      string
}

func (t problemTest) OpenInput() (*os.File, error) {
	return os.Open(t.inputPath)
}

func (t problemTest) OpenAnswer() (*os.File, error) {
	return os.Open(t.answerPath)
}

func (t problemTest) Points() float64 {
	return t.points
}

func (t problemTest) Group() string {
	return t.group
}

type polygonProblemStatement struct {
	problem  *polygonProblem
	language string
}

func (s *polygonProblemStatement) Locale() string {
	return polygonLocales[s.language]
}

func (s *polygonProblemStatement) GetConfig() (models.ProblemStatementConfig, error) {
	statement, err := polygon.ReadProblemStatementConfig(filepath.Join(
		s.problem.path, "statements", s.language, "problem-properties.json",
	))
	if err != nil {
		return models.ProblemStatementConfig{}, err
	}
	config := models.ProblemStatementConfig{
		Locale:      s.Locale(),
		Title:       statement.Name,
		Legend:      statement.Legend,
		Input:       statement.Input,
		Output:      statement.Output,
		Notes:       statement.Notes,
		Scoring:     statement.Scoring,
		Interaction: statement.Interaction,
	}
	for _, sample := range statement.SampleTests {
		config.Samples = append(
			config.Samples,
			models.ProblemStatementSample{
				Input:  sample.Input,
				Output: sample.Output,
			},
		)
	}
	return config, nil
}

func (s *polygonProblemStatement) GetResources() ([]ProblemResource, error) {
	config, err := s.GetConfig()
	if err != nil {
		return nil, err
	}
	resourcesDir := filepath.Join(s.problem.path, "statements", s.language)
	files, err := os.ReadDir(resourcesDir)
	if err != nil {
		return nil, err
	}
	resources := []ProblemResource{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		inStatements := strings.Contains(config.Title, name) ||
			strings.Contains(config.Legend, name) ||
			strings.Contains(config.Input, name) ||
			strings.Contains(config.Output, name) ||
			strings.Contains(config.Notes, name)
		if !inStatements {
			continue
		}
		resources = append(resources, polygonProblemResource{
			path: filepath.Join(resourcesDir, name),
			name: name,
		})
	}
	return resources, nil
}

type polygonProblemResource struct {
	path string
	name string
}

func (p polygonProblemResource) Name() string {
	return p.name
}

func (p polygonProblemResource) GetMD5() (string, error) {
	file, err := os.Open(p.path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	md5, _, err := hash.CalculateMD5(file)
	return md5, err
}

func (p polygonProblemResource) Open() (*os.File, error) {
	return os.Open(p.path)
}

var polygonLocales = map[string]string{
	"russian": "ru",
	"english": "en",
}

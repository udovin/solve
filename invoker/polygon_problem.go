package invoker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/udovin/algo/futures"
	"github.com/udovin/solve/models"
	"github.com/udovin/solve/pkg/archives"
	"github.com/udovin/solve/pkg/hash"
	"github.com/udovin/solve/pkg/logs"
	"github.com/udovin/solve/pkg/polygon"
)

func extractPolygonProblem(
	source, target string, compilers *compilerManager,
) (Problem, error) {
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
		path:      target,
		config:    config,
		compilers: compilers,
	}, nil
}

type compiled struct {
	path     string
	compiler Compiler
}

type polygonProblem struct {
	path        string
	config      polygon.Problem
	compilers   *compilerManager
	executables map[string]compiled
}

func (p *polygonProblem) compileExecutable(
	ctx context.Context, polygonType string, source string, resources []MountFile,
) (compiled, error) {
	polygonName := "polygon." + polygonType
	compilerName, err := p.compilers.GetCompilerName(polygonName)
	if err != nil {
		return compiled{}, err
	}
	compiler, err := p.compilers.GetCompiler(ctx, compilerName)
	if err != nil {
		return compiled{}, err
	}
	target := strings.TrimSuffix(source, filepath.Ext(source))
	if _, ok := p.executables[target]; !ok {
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
			return compiled{}, err
		}
		if !report.Success() {
			return compiled{}, fmt.Errorf(
				"cannot compile %q with compiler %q: %q",
				source, compilerName, report.Log,
			)
		}
		p.compilers.logger.Debug(
			"Compiled executable",
			logs.Any("path", source),
		)
		p.executables[target] = compiled{
			path:     targetPath,
			compiler: compiler,
		}
	}
	return p.executables[target], nil
}

func (p *polygonProblem) generateTest(ctx context.Context, args []string, input string) error {
	if len(args) == 0 {
		return fmt.Errorf("cannot find executable")
	}
	executable, ok := p.executables[fmt.Sprintf("files/%s", args[0])]
	if !ok {
		return fmt.Errorf("cannot find executable: %q", args[0])
	}
	report, err := executable.compiler.Execute(ctx, ExecuteOptions{
		Binary: executable.path,
		Args:   args[1:],
		OutputFiles: []MountFile{
			{Source: filepath.Join(p.path, input), Target: "stdout"},
		},
		TimeLimit:   20 * time.Second,
		MemoryLimit: 256 * 1024 * 1024,
	})
	if err != nil {
		return fmt.Errorf("cannot execute generator %q: %w", args[0], err)
	}
	if !report.Success() {
		return fmt.Errorf("generator exited with code: %v", report.ExitCode)
	}
	return nil
}

func (p *polygonProblem) Compile(ctx context.Context) error {
	p.executables = map[string]compiled{}
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
			ctx, e.Source.Type, e.Source.Path, resources,
		); err != nil {
			return err
		}
	}
	var interactor *compiled
	if p.config.Assets != nil {
		if e := p.config.Assets.Checker; e != nil {
			if _, err := p.compileExecutable(
				ctx, e.Source.Type, e.Source.Path, resources,
			); err != nil {
				return err
			}
		}
		if e := p.config.Assets.Interactor; e != nil {
			if c, err := p.compileExecutable(
				ctx, e.Source.Type, e.Source.Path, resources,
			); err != nil {
				return err
			} else {
				interactor = &c
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
	var solution compiled
	{
		polygonName := "polygon." + mainSolution.Source.Type
		compilerName, err := p.compilers.GetCompilerName(polygonName)
		if err != nil {
			return err
		}
		compiler, err := p.compilers.GetCompiler(ctx, compilerName)
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
		p.compilers.logger.Debug(
			"Compiled solution",
			logs.Any("path", mainSolution.Source.Path),
		)
		solution = compiled{
			path:     targetPath,
			compiler: compiler,
		}
	}
	for _, testSet := range p.config.TestSets {
		for i, test := range testSet.Tests {
			input := fmt.Sprintf(testSet.InputPathPattern, i+1)
			answer := fmt.Sprintf(testSet.AnswerPathPattern, i+1)
			if test.Cmd != "" {
				if err := p.generateTest(
					ctx, strings.Fields(test.Cmd), input,
				); err != nil {
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
					interactorProcess, err := interactor.compiler.PrepareExecute(ctx, ExecuteOptions{
						Binary: interactor.path,
						Args:   []string{"input.in", "output.out"},
						Stdin:  solutionReader,
						Stdout: interactorWriter,
						InputFiles: []MountFile{
							{Source: filepath.Join(p.path, input), Target: "input.in"},
						},
						OutputFiles: []MountFile{
							{Source: filepath.Join(p.path, answer), Target: "output.out"},
						},
						TimeLimit:   2 * time.Duration(testSet.TimeLimit) * time.Millisecond,
						MemoryLimit: 256 * 1024 * 1024,
					})
					if err != nil {
						return fmt.Errorf("cannot prepare interactor: %w", err)
					}
					defer func() { _ = interactorProcess.Release() }()
					solutionProcess, err := solution.compiler.PrepareExecute(ctx, ExecuteOptions{
						Binary:      solution.path,
						Stdin:       interactorReader,
						Stdout:      solutionWriter,
						TimeLimit:   time.Duration(testSet.TimeLimit) * time.Millisecond,
						MemoryLimit: testSet.MemoryLimit,
					})
					if err != nil {
						return fmt.Errorf("cannot prepare solution: %w", err)
					}
					defer func() { _ = solutionProcess.Release() }()
					if err := interactorProcess.Start(); err != nil {
						return fmt.Errorf("cannot execute interactor: %w", err)
					}
					if err := solutionProcess.Start(); err != nil {
						return fmt.Errorf("cannot execute solution: %w", err)
					}
					interactorReportFuture := futures.Call(func() (ExecuteReport, error) {
						defer func() {
							_ = solutionReader.Close()
							_ = interactorWriter.Close()
						}()
						return interactorProcess.Wait()
					})
					solutionReportFuture := futures.Call(func() (ExecuteReport, error) {
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
					if !solutionReport.Success() {
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
				report, err := solution.compiler.Execute(ctx, ExecuteOptions{
					Binary: solution.path,
					InputFiles: []MountFile{
						{Source: filepath.Join(p.path, input), Target: "stdin"},
					},
					OutputFiles: []MountFile{
						{Source: filepath.Join(p.path, answer), Target: "stdout"},
					},
					TimeLimit:   time.Duration(testSet.TimeLimit) * time.Millisecond,
					MemoryLimit: testSet.MemoryLimit,
				})
				if err != nil {
					return fmt.Errorf("cannot execute solution: %w", err)
				}
				if !report.Success() {
					return fmt.Errorf("solution exited with code: %v", report.ExitCode)
				}
			}
			p.compilers.logger.Debug(
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
		polygonName := "polygon." + checker.Source.Type
		compilerName, err := p.compilers.GetCompilerName(polygonName)
		if err != nil {
			return nil, err
		}
		source := checker.Source.Path
		target := strings.TrimSuffix(source, filepath.Ext(source))
		targetPath := filepath.Join(p.path, target)
		executables = append(executables, problemExecutable{
			name:       "checker",
			kind:       TestlibChecker,
			binaryPath: targetPath,
			compiler:   compilerName,
		})
	}
	if p.config.Assets.Interactor != nil {
		interactor := p.config.Assets.Interactor
		polygonName := "polygon." + interactor.Source.Type
		compilerName, err := p.compilers.GetCompilerName(polygonName)
		if err != nil {
			return nil, err
		}
		source := interactor.Source.Path
		target := strings.TrimSuffix(source, filepath.Ext(source))
		targetPath := filepath.Join(p.path, target)
		executables = append(executables, problemExecutable{
			name:       "interactor",
			kind:       TestlibInteractor,
			binaryPath: targetPath,
			compiler:   compilerName,
		})
	}
	return executables, nil
}

type problemExecutable struct {
	name       string
	kind       ProblemExecutableKind
	binaryPath string
	compiler   string
}

func (e problemExecutable) Name() string {
	return e.name
}

func (e problemExecutable) Kind() ProblemExecutableKind {
	return e.kind
}

func (e problemExecutable) Compiler() string {
	return e.compiler
}

func (e problemExecutable) OpenBinary() (*os.File, error) {
	return os.Open(e.binaryPath)
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

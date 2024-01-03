package cache

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/udovin/solve/internal/pkg/archives"
	"github.com/udovin/solve/internal/pkg/compilers"
	"github.com/udovin/solve/internal/pkg/problems"
)

type problemTestConfig struct {
	Input  string  `json:"input"`
	Answer string  `json:"answer"`
	Points float64 `json:"points,omitempty"`
	Group  string  `json:"group,omitempty"`
}

type problemTestGroupConfig struct {
	Name         string `json:"name"`
	PointsPolicy string `json:"points_policy"`
}

type problemTestSetConfig struct {
	Name        string                   `json:"name"`
	Dir         string                   `json:"dir"`
	Tests       []problemTestConfig      `json:"tests"`
	TimeLimit   int64                    `json:"time_limit,omitempty"`
	MemoryLimit int64                    `json:"memory_limit,omitempty"`
	Groups      []problemTestGroupConfig `json:"groups,omitempty"`
}

type problemExecutableConfig struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Binary   string `json:"binary"`
	Compiler string `json:"compiler"`
}

type problemConfig struct {
	Version     string                    `json:"version"`
	Executables []problemExecutableConfig `json:"executables,omitempty"`
	TestSets    []problemTestSetConfig    `json:"test_sets,omitempty"`
	// Deprecated.
	DeprecatedTestGroups []problemTestSetConfig `json:"test_groups,omitempty"`
}

const problemConfigVersion = "0.1"

func writeZipDirectory(writer *zip.Writer, name string) error {
	header := zip.FileHeader{
		Name:   name + "/",
		Method: zip.Deflate,
	}
	header.SetMode(fs.ModePerm)
	_, err := writer.CreateHeader(&header)
	return err
}

func BuildCompiledProblem(ctx context.Context, compilers problems.CompileContext, problem problems.Problem, target string) error {
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	writer := zip.NewWriter(file)
	defer func() { _ = writer.Close() }()
	config := problemConfig{Version: problemConfigVersion}
	executables, err := problem.GetExecutables()
	if err != nil {
		return err
	}
	if err := writeZipDirectory(writer, "executables"); err != nil {
		return err
	}
	for _, executable := range executables {
		compiler, err := executable.GetCompiler(ctx, compilers)
		if err != nil {
			return err
		}
		executableConfig := problemExecutableConfig{
			Name:     executable.Name(),
			Kind:     string(executable.Kind()),
			Binary:   path.Join("executables", path.Base(executable.Name())),
			Compiler: compiler.Name(),
		}
		if err := func() error {
			binaryFile, err := executable.OpenBinary()
			if err != nil {
				return err
			}
			defer func() { _ = binaryFile.Close() }()
			binaryHeader := zip.FileHeader{
				Name:   executableConfig.Binary,
				Method: zip.Deflate,
			}
			binaryHeader.SetMode(fs.ModePerm)
			header, err := writer.CreateHeader(&binaryHeader)
			if err != nil {
				return err
			}
			_, err = io.Copy(header, binaryFile)
			return err
		}(); err != nil {
			return err
		}
		config.Executables = append(config.Executables, executableConfig)
	}
	testSets, err := problem.GetTestSets()
	if err != nil {
		return err
	}
	if err := writeZipDirectory(writer, "tests"); err != nil {
		return err
	}
	for i, testSet := range testSets {
		tests, err := testSet.GetTests()
		if err != nil {
			return err
		}
		name := testSet.Name()
		if name == "" {
			name = fmt.Sprintf("tests%d", i+1)
		}
		testSetConfig := problemTestSetConfig{
			Name:        name,
			Dir:         path.Join("tests", name),
			TimeLimit:   testSet.TimeLimit(),
			MemoryLimit: testSet.MemoryLimit(),
		}
		if err := writeZipDirectory(writer, testSetConfig.Dir); err != nil {
			return err
		}
		testNameFmt := "%d"
		if len(tests) >= 100 {
			testNameFmt = "%03d"
		} else if len(tests) >= 10 {
			testNameFmt = "%02d"
		}
		for j, test := range tests {
			testName := fmt.Sprintf(testNameFmt, j+1)
			testConfig := problemTestConfig{
				Input:  testName + ".in",
				Answer: testName + ".ans",
				Points: test.Points(),
				Group:  test.Group(),
			}
			if err := func() error {
				inputFile, err := test.OpenInput()
				if err != nil {
					return err
				}
				defer func() { _ = inputFile.Close() }()
				header, err := writer.Create(
					path.Join(testSetConfig.Dir, testConfig.Input),
				)
				if err != nil {
					return err
				}
				_, err = io.Copy(header, inputFile)
				return err
			}(); err != nil {
				return err
			}
			if err := func() error {
				answerFile, err := test.OpenAnswer()
				if err != nil {
					return err
				}
				defer func() { _ = answerFile.Close() }()
				header, err := writer.Create(
					path.Join(testSetConfig.Dir, testConfig.Answer),
				)
				if err != nil {
					return err
				}
				_, err = io.Copy(header, answerFile)
				return err
			}(); err != nil {
				return err
			}
			testSetConfig.Tests = append(testSetConfig.Tests, testConfig)
		}
		groups, err := testSet.GetGroups()
		if err != nil {
			return err
		}
		for _, group := range groups {
			testSetConfig.Groups = append(testSetConfig.Groups, problemTestGroupConfig{
				Name:         group.Name(),
				PointsPolicy: string(group.PointsPolicy()),
			})
		}
		config.TestSets = append(config.TestSets, testSetConfig)
	}
	header, err := writer.Create("problem.json")
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(header)
	encoder.SetIndent("", "\t")
	if err := encoder.Encode(config); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return file.Sync()
}

func extractCompiledProblem(
	source, target string,
) (problems.Problem, error) {
	if err := archives.ExtractZip(source, target); err != nil {
		return nil, fmt.Errorf("cannot extract problem: %w", err)
	}
	var config problemConfig
	if err := func() error {
		file, err := os.Open(filepath.Join(target, "problem.json"))
		if err != nil {
			return fmt.Errorf("cannot read problem config: %w", err)
		}
		defer func() { _ = file.Close() }()
		return json.NewDecoder(file).Decode(&config)
	}(); err != nil {
		_ = os.RemoveAll(target)
		return nil, err
	}
	// Fix deprecated fields.
	if config.DeprecatedTestGroups != nil {
		config.TestSets = append(config.TestSets, config.DeprecatedTestGroups...)
		config.DeprecatedTestGroups = nil
	}
	problem := compiledProblem{
		path:   target,
		config: config,
	}
	return &problem, nil
}

type compiledProblem struct {
	path   string
	config problemConfig
}

func (p *compiledProblem) Compile(ctx context.Context, compilers problems.CompileContext) error {
	return nil
}

func (p *compiledProblem) GetExecutables() ([]problems.ProblemExecutable, error) {
	var executables []problems.ProblemExecutable
	for _, executable := range p.config.Executables {
		executables = append(executables, compiledProblemExecutable{
			path:   p.path,
			config: executable,
		})
	}
	return executables, nil
}

func (p *compiledProblem) GetTestSets() ([]problems.ProblemTestSet, error) {
	var groups []problems.ProblemTestSet
	for _, group := range p.config.TestSets {
		groups = append(groups, &compiledProblemTestSet{
			path:   filepath.Join(p.path, group.Dir),
			config: group,
		})
	}
	return groups, nil
}

func (p *compiledProblem) GetStatements() ([]problems.ProblemStatement, error) {
	return nil, nil
}

type compiledProblemExecutable struct {
	path   string
	config problemExecutableConfig
}

func (e compiledProblemExecutable) Name() string {
	return e.config.Name
}

func (e compiledProblemExecutable) Kind() problems.ProblemExecutableKind {
	return problems.ProblemExecutableKind(e.config.Kind)
}

func (e compiledProblemExecutable) OpenBinary() (*os.File, error) {
	return os.Open(filepath.Join(e.path, e.config.Binary))
}

func (e compiledProblemExecutable) GetCompiler(ctx context.Context, compilers problems.CompileContext) (compilers.Compiler, error) {
	return compilers.GetCompiler(ctx, e.config.Compiler)
}

type compiledProblemTestSet struct {
	path   string
	config problemTestSetConfig
}

func (g *compiledProblemTestSet) Name() string {
	return g.config.Name
}

func (g *compiledProblemTestSet) TimeLimit() int64 {
	return g.config.TimeLimit
}

func (g *compiledProblemTestSet) MemoryLimit() int64 {
	return g.config.MemoryLimit
}

func (g *compiledProblemTestSet) GetGroups() ([]problems.ProblemTestGroup, error) {
	var groups []problems.ProblemTestGroup
	for _, group := range g.config.Groups {
		groups = append(groups, problemTestGroup{
			name:         group.Name,
			pointsPolicy: problems.ProblemPointsPolicy(group.PointsPolicy),
		})
	}
	return groups, nil
}

func (g *compiledProblemTestSet) GetTests() ([]problems.ProblemTest, error) {
	var tests []problems.ProblemTest
	for _, test := range g.config.Tests {
		tests = append(tests, problemTest{
			inputPath:  filepath.Join(g.path, test.Input),
			answerPath: filepath.Join(g.path, test.Answer),
			points:     test.Points,
			group:      test.Group,
		})
	}
	return tests, nil
}

var _ problems.Problem = (*compiledProblem)(nil)

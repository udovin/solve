package invoker

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/udovin/solve/pkg"
)

type problemTestConfig struct {
	Input  string `json:"input"`
	Answer string `json:"answer"`
}

type problemTestGroupConfig struct {
	Name        string              `json:"name"`
	Dir         string              `json:"dir"`
	Tests       []problemTestConfig `json:"tests"`
	TimeLimit   int64               `json:"time_limit,omitempty"`
	MemoryLimit int64               `json:"memory_limit,omitempty"`
}

type problemExecutableConfig struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Binary   string `json:"binary"`
	Compiler string `json:"compiler"`
}

type problemConfig struct {
	Version     string                    `json:"version"`
	Executables []problemExecutableConfig `json:"executables"`
	TestGroups  []problemTestGroupConfig  `json:"test_groups"`
}

const problemConfigVersion = "0.1"

func buildCompiledProblem(problem Problem, target string) error {
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
	if _, err := writer.Create("executables/"); err != nil {
		return err
	}
	for _, executable := range executables {
		executableConfig := problemExecutableConfig{
			Name:     executable.Name(),
			Kind:     string(executable.Kind()),
			Binary:   path.Join("executables", path.Base(executable.Name())),
			Compiler: executable.Compiler(),
		}
		if err := func() error {
			binaryFile, err := executable.OpenBinary()
			if err != nil {
				return err
			}
			defer func() { _ = binaryFile.Close() }()
			header, err := writer.Create(executableConfig.Binary)
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
	groups, err := problem.GetTestGroups()
	if err != nil {
		return err
	}
	if _, err := writer.Create("groups/"); err != nil {
		return err
	}
	for i, group := range groups {
		tests, err := group.GetTests()
		if err != nil {
			return err
		}
		name := group.Name()
		if name == "" {
			name = fmt.Sprintf("group%d", i+1)
		}
		groupConfig := problemTestGroupConfig{
			Name:        name,
			Dir:         path.Join("groups", name),
			TimeLimit:   group.TimeLimit(),
			MemoryLimit: group.MemoryLimit(),
		}
		if _, err := writer.Create(groupConfig.Dir + "/"); err != nil {
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
			}
			if err := func() error {
				inputFile, err := test.OpenInput()
				if err != nil {
					return err
				}
				defer func() { _ = inputFile.Close() }()
				header, err := writer.Create(
					path.Join(groupConfig.Dir, testConfig.Input),
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
					path.Join(groupConfig.Dir, testConfig.Answer),
				)
				if err != nil {
					return err
				}
				_, err = io.Copy(header, answerFile)
				return err
			}(); err != nil {
				return err
			}
			groupConfig.Tests = append(groupConfig.Tests, testConfig)
		}
		config.TestGroups = append(config.TestGroups, groupConfig)
	}
	header, err := writer.Create("problem.json")
	if err != nil {
		return err
	}
	return json.NewEncoder(header).Encode(config)
}

func extractCompiledProblem(
	source, target string, compilers *compilerManager,
) (Problem, error) {
	if err := pkg.ExtractZip(source, target); err != nil {
		return nil, fmt.Errorf("cannot extract problem: %w", err)
	}
	var config problemConfig
	if err := func() error {
		file, err := os.Open(filepath.Join(target, "problem.json"))
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		return json.NewDecoder(file).Decode(&config)
	}(); err != nil {
		_ = os.RemoveAll(target)
		return nil, err
	}
	problem := compiledProblem{
		path:      target,
		compilers: compilers,
		config:    config,
	}
	return &problem, nil
}

type compiledProblem struct {
	path      string
	compilers *compilerManager
	config    problemConfig
}

func (p *compiledProblem) Compile(ctx context.Context) error {
	return nil
}

func (p *compiledProblem) GetExecutables() ([]ProblemExecutable, error) {
	var executables []ProblemExecutable
	for _, executable := range p.config.Executables {
		executables = append(executables, problemExecutable{
			name:       executable.Name,
			kind:       ProblemExecutableKind(executable.Kind),
			binaryPath: executable.Binary,
			compiler:   executable.Compiler,
		})
	}
	return executables, nil
}

func (p *compiledProblem) GetTestGroups() ([]ProblemTestGroup, error) {
	var groups []ProblemTestGroup
	for _, group := range p.config.TestGroups {
		groups = append(groups, &compiledProblemTestGroup{
			path:   filepath.Join(p.path, group.Dir),
			config: group,
		})
	}
	return groups, nil
}

func (p *compiledProblem) GetStatements() ([]ProblemStatement, error) {
	return nil, nil
}

type compiledProblemTestGroup struct {
	path   string
	config problemTestGroupConfig
}

func (g *compiledProblemTestGroup) Name() string {
	return g.config.Name
}

func (g *compiledProblemTestGroup) TimeLimit() int64 {
	return g.config.TimeLimit
}

func (g *compiledProblemTestGroup) MemoryLimit() int64 {
	return g.config.MemoryLimit
}

func (g *compiledProblemTestGroup) GetTests() ([]ProblemTest, error) {
	var tests []ProblemTest
	for _, test := range g.config.Tests {
		tests = append(tests, problemTest{
			inputPath:  filepath.Join(g.path, test.Input),
			answerPath: filepath.Join(g.path, test.Answer),
		})
	}
	return tests, nil
}

var _ Problem = (*compiledProblem)(nil)

package polygon

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"path/filepath"
)

// Name represents problem name.
type Name struct {
	Language string `xml:"language,attr"`
	Value    string `xml:"value,attr"`
}

// Statement represents problem statement.
type Statement struct {
	Language string `xml:"language,attr"`
	Charset  string `xml:"charset,attr"`
	Path     string `xml:"path,attr"`
	MathJax  bool   `xml:"mathjax,attr"`
	Type     string `xml:"type,attr"`
}

// Test represents test.
type Test struct {
	Method string `xml:"method,attr"`
	Sample bool   `xml:"sample,attr"`
	Cmd    string `xml:"cmd,attr"`
}

// TestSet represents a group of tests.
type TestSet struct {
	TimeLimit         int64  `xml:"time-limit"`
	MemoryLimit       int64  `xml:"memory-limit"`
	TestCount         int    `xml:"test-count"`
	InputPathPattern  string `xml:"input-path-pattern"`
	AnswerPathPattern string `xml:"answer-path-pattern"`
	Tests             []Test `xml:"tests>test"`
}

// Problem represents a problem.
type Problem struct {
	Names      []Name      `xml:"names>name"`
	Statements []Statement `xml:"statements>statement"`
	TestSets   []TestSet   `xml:"judging>testset"`
}

const (
	configPath            = "problem.xml"
	statementsDir         = "statements"
	problemPropertiesPath = "problem-properties.json"
)

// ReadProblem reads problem from directory.
func ReadProblem(dir string) (Problem, error) {
	data, err := ioutil.ReadFile(filepath.Join(dir, configPath))
	if err != nil {
		return Problem{}, err
	}
	var problem Problem
	if err := xml.Unmarshal(data, &problem); err != nil {
		return Problem{}, err
	}
	return problem, nil
}

type SampleTest struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type ProblemProperties struct {
	Name        string       `json:"name"`
	Legend      string       `json:"legend"`
	Input       string       `json:"input"`
	Output      string       `json:"output"`
	Notes       string       `json:"notes"`
	Tutorial    string       `json:"tutorial"`
	TimeLimit   int          `json:"timeLimit"`
	MemoryLimit int64        `json:"memoryLimit"`
	InputFile   string       `json:"inputFile"`
	OutputFile  string       `json:"outputFile"`
	SampleTests []SampleTest `json:"sampleTests"`
}

func ReadProblemProperites(
	dir string, language string,
) (ProblemProperties, error) {
	data, err := ioutil.ReadFile(filepath.Join(
		dir, statementsDir, language, problemPropertiesPath,
	))
	if err != nil {
		return ProblemProperties{}, err
	}
	var properties ProblemProperties
	if err := json.Unmarshal(data, &properties); err != nil {
		return ProblemProperties{}, err
	}
	return properties, nil
}

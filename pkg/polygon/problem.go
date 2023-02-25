package polygon

import (
	"encoding/json"
	"encoding/xml"
	"os"
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
	Method string  `xml:"method,attr"`
	Sample bool    `xml:"sample,attr"`
	Cmd    string  `xml:"cmd,attr"`
	Points float64 `xml:"points,attr"`
	Group  string  `xml:"group,attr"`
}

type Group struct {
	Name           string  `xml:"name,attr"`
	Points         float64 `xml:"points,attr"`
	PointsPolicy   string  `xml:"points-policy,attr"`
	FeedbackPolicy string  `xml:"feedback-policy,attr"`
}

// TestSet represents a group of tests.
type TestSet struct {
	Name              string  `xml:"name,attr"`
	TimeLimit         int64   `xml:"time-limit"`
	MemoryLimit       int64   `xml:"memory-limit"`
	TestCount         int     `xml:"test-count"`
	InputPathPattern  string  `xml:"input-path-pattern"`
	AnswerPathPattern string  `xml:"answer-path-pattern"`
	Tests             []Test  `xml:"tests>test"`
	Groups            []Group `xml:"groups>group"`
}

type Resource struct {
	Path string `xml:"path,attr"`
	Type string `xml:"type,attr"`
}

type Checker struct {
	Name   string    `xml:"name,attr"`
	Type   string    `xml:"type,attr"`
	Source *Resource `xml:"source"`
	Binary *Resource `xml:"binary"`
}

type Solution struct {
	Tag    string    `xml:"tag,attr"`
	Source *Resource `xml:"source"`
	Binary *Resource `xml:"binary"`
}

type ProblemAssets struct {
	Checker   *Checker   `xml:"checker"`
	Solutions []Solution `xml:"solutions>solution"`
}

type Executable struct {
	Source *Resource `xml:"source"`
	Binary *Resource `xml:"binary"`
}

type ProblemFiles struct {
	Resources   []Resource   `xml:"resources>file"`
	Executables []Executable `xml:"executables>executable"`
}

// Problem represents a problem.
type Problem struct {
	Names      []Name         `xml:"names>name"`
	Statements []Statement    `xml:"statements>statement"`
	TestSets   []TestSet      `xml:"judging>testset"`
	Assets     *ProblemAssets `xml:"assets"`
	Files      *ProblemFiles  `xml:"files"`
}

// ReadProblemConfig reads problem config from file.
func ReadProblemConfig(path string) (Problem, error) {
	data, err := os.ReadFile(path)
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

type ProblemStatementConfig struct {
	Name        string       `json:"name"`
	Legend      string       `json:"legend"`
	Input       string       `json:"input"`
	Output      string       `json:"output"`
	Notes       string       `json:"notes"`
	Scoring     string       `json:"scoring"`
	Tutorial    string       `json:"tutorial"`
	TimeLimit   int          `json:"timeLimit"`
	MemoryLimit int64        `json:"memoryLimit"`
	InputFile   string       `json:"inputFile"`
	OutputFile  string       `json:"outputFile"`
	SampleTests []SampleTest `json:"sampleTests"`
}

func ReadProblemStatementConfig(path string) (ProblemStatementConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProblemStatementConfig{}, err
	}
	var properties ProblemStatementConfig
	if err := json.Unmarshal(data, &properties); err != nil {
		return ProblemStatementConfig{}, err
	}
	return properties, nil
}

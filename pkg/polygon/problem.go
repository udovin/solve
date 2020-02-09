package polygon

import (
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

const configPath = "problem.xml"

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

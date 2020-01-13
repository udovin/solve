package pkg

import (
	"io"
)

// Checker represents checker.
type Checker interface {
}

// Validator represents validator.
type Validator interface {
}

// Generator represents generator.
type Generator interface {
}

// Test represents simple test case.
type Test interface {
	// Input should return reader for input file.
	Input() (io.ReadCloser, error)
	// Answer should return reader for answer file.
	Answer() (io.ReadCloser, error)
	// Checker should return checker.
	Checker() (Checker, error)
	// Validator represents validator.
	Validator() (Validator, error)
	// Generator represents generator.
	Generator() (Generator, error)
}

// TestGroup represents group of tests.
type TestGroup interface {
	// IsSample should return true if test group contains sample tests.
	IsSample() (bool, error)
	// IsPreliminary should return true if test group
	// contains preliminary tests.
	IsPreliminary() (bool, error)
	// Tests should return a list of tests in group.
	Tests() ([]Test, error)
}

// Statement represents a statement.
type Statement interface {
	// Language should return language of problem.
	Language() (string, error)
	// Title should return title of problem.
	Title() (string, error)
	// Description should return description of problem.
	Description() (string, error)
}

// Problem represents a problem.
type Problem interface {
	// Statements should return list of problem statements.
	Statements() ([]Statement, error)
	// TestGroups should return list of problem test groups.
	TestGroups() ([]TestGroup, error)
}

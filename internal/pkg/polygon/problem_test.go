package polygon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/udovin/solve/internal/pkg/archives"
)

const testDataDir = "../../../testdata"

func TestProblem(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "problem")
	if err := archives.ExtractZip(
		filepath.Join(testDataDir, "a-plus-b.zip"),
		dir,
	); err != nil {
		t.Fatal("Error:", err)
	}
	problem, err := ReadProblemConfig(filepath.Join(dir, "problem.xml"))
	if err != nil {
		t.Fatal(err)
	}
	_ = problem
	properties, err := ReadProblemStatementConfig(filepath.Join(
		dir, "statements", "english", "problem-properties.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	_ = properties
}

func TestNotFoundProblem(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "not-found-problem")
	if _, err := ReadProblemConfig(
		filepath.Join(dir, "problem.xml"),
	); err == nil {
		t.Fatal("Expected error")
	}
}

func TestInvalidProblem(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "invalid-problem")
	if err := archives.ExtractZip(
		filepath.Join(testDataDir, "a-plus-b.zip"),
		dir,
	); err != nil {
		t.Fatal("Error:", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "problem.xml"), []byte("><"), 0644,
	); err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := ReadProblemConfig(
		filepath.Join(dir, "problem.xml"),
	); err == nil {
		t.Fatal("Expected error")
	}
}

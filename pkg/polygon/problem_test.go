package polygon

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/udovin/solve/pkg"
)

func TestProblem(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "problem")
	if err := pkg.ExtractZip(
		filepath.Join("../../testdata", "a-plus-b.zip"),
		dir,
	); err != nil {
		t.Fatal("Error:", err)
	}
	problem, err := ReadProblem(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = problem
}

func TestNotFoundProblem(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "not-found-problem")
	if _, err := ReadProblem(dir); err == nil {
		t.Fatal("Expected error")
	}
}

func TestInvalidProblem(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "invalid-problem")
	if err := pkg.ExtractZip(
		filepath.Join("../../testdata", "a-plus-b.zip"),
		dir,
	); err != nil {
		t.Fatal("Error:", err)
	}
	if err := ioutil.WriteFile(
		filepath.Join(dir, "problem.xml"), []byte("><"), 0644,
	); err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := ReadProblem(dir); err == nil {
		t.Fatal("Expected error")
	}
}

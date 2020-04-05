package polygon

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/udovin/solve/pkg"
)

const testPrefix = "test-"

func extractTestPackage(t testing.TB, source string) string {
	target, err := ioutil.TempDir("", testPrefix)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := pkg.ExtractZip(
		filepath.Join("testdata", source), target,
	); err != nil {
		_ = os.RemoveAll(target)
		t.Fatal("Error:", err)
	}
	return target
}

func TestProblem(t *testing.T) {
	dir := extractTestPackage(t, "a-plus-b.zip")
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	problem, err := ReadProblem(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = problem
}

func TestProblem_NotFound(t *testing.T) {
	if _, err := ReadProblem("not-found"); err == nil {
		t.Fatal("Expected error")
	}
}

func TestProblem_Invalid(t *testing.T) {
	dir := extractTestPackage(t, "a-plus-b.zip")
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	if err := ioutil.WriteFile(
		filepath.Join(dir, "problem.xml"), []byte("><"), 0644,
	); err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := ReadProblem(dir); err == nil {
		t.Fatal("Expected error")
	}
}

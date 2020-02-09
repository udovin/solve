package polygon

import (
	"io/ioutil"
	"log"
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
	log.Println(problem)
}

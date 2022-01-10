package pkg

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const testPrefix = "test-"

func TestTarGz(t *testing.T) {
	target, err := ioutil.TempDir("", testPrefix)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if err := ExtractTarGz(
		filepath.Join("testdata", "alpine.tar.gz"), target,
	); err != nil {
		_ = os.RemoveAll(target)
		t.Fatal("Error:", err)
	}
	defer func() {
		_ = os.RemoveAll(target)
	}()
}

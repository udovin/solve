package pkg

import (
	"path/filepath"
	"testing"
)

const testPrefix = "test-"

func TestTarGz(t *testing.T) {
	if err := ExtractTarGz(
		filepath.Join("testdata", "alpine.tar.gz"),
		filepath.Join(t.TempDir(), "alpine"),
	); err != nil {
		t.Fatal("Error:", err)
	}
}

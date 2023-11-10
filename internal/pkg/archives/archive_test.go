package archives

import (
	"path/filepath"
	"testing"
)

const testDataDir = "../../../testdata"

func TestTarGz(t *testing.T) {
	if err := ExtractTarGz(
		filepath.Join(testDataDir, "alpine-cpp.tar.gz"),
		filepath.Join(t.TempDir(), "alpine"),
	); err != nil {
		t.Fatal("Error:", err)
	}
}

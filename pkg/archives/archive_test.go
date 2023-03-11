package archives

import (
	"path/filepath"
	"testing"
)

func TestTarGz(t *testing.T) {
	if err := ExtractTarGz(
		filepath.Join("../../testdata", "alpine-cpp.tar.gz"),
		filepath.Join(t.TempDir(), "alpine"),
	); err != nil {
		t.Fatal("Error:", err)
	}
}

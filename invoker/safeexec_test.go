package invoker

import (
	"testing"
	"path/filepath"
	"context"

	"github.com/udovin/solve/pkg"
)

func TestSafeexec(t *testing.T) {
	safeexecPath := filepath.Join(t.TempDir(), "safeexec")
	alpinePath := filepath.Join(t.TempDir(), "alpine")
	if err := pkg.ExtractTarGz(
		filepath.Join("../testdata", "alpine.tar.gz"),
		alpinePath,
	); err != nil {
		t.Fatal("Error:", err)
	}
	safeexec, err := newSafeexecProcessor("../safeexec/safeexec", safeexecPath, "solve-safeexec")
	if err != nil {
		t.Fatal("Error:", err)
	}
	processConfig := safeexecProcessConfig{
		ImagePath: alpinePath,
		Command:   []string{"/bin/sh", "-c", "echo -n 'solve_test'"},
	}
	process, err := safeexec.Execute(context.Background(), processConfig)
	if err != nil {
		t.Fatal("Error:", err)
	}
	defer func() { _ = process.Release() }()
}

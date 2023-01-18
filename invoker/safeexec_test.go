package invoker

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	stdout := strings.Builder{}
	processConfig := safeexecProcessConfig{
		ImagePath:   alpinePath,
		Command:     []string{"/bin/sh", "-c", "echo -n 'solve_test'"},
		TimeLimit:   time.Second,
		MemoryLimit: 1024 * 1024,
		Stdout:      &stdout,
	}
	process, err := safeexec.Create(context.Background(), processConfig)
	if err != nil {
		t.Fatal("Error:", err)
	}
	defer func() { _ = process.Release() }()
	if err := process.Start(); err != nil {
		t.Fatal("Error:", err)
	}
	t.Log(process.path)
	t.Log(process.cgroupPath)
	if err := process.Wait(); err != nil {
		t.Fatal("Error:", err)
	}
	if s := stdout.String(); s != "solve_test" {
		t.Fatal("Expected:", "solve_test", "got:", s)
	}
}

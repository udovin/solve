package invoker

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/udovin/solve/pkg"
)

func TestSafeexecSimple(t *testing.T) {
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
		Layers:      []string{alpinePath},
		Command:     []string{"/bin/sh", "-c", "sleep 1 && echo -n 'solve_test'"},
		TimeLimit:   2 * time.Second,
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
	report, err := process.Wait()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if report.ExitCode != 0 {
		t.Fatal("Exit code:", report.ExitCode)
	}
	if report.Memory <= 0 {
		t.Fatal("Invalid memory:", report.Memory)
	}
	if report.Time < time.Second || report.Time > 3*time.Second {
		t.Fatal("Invalid time:", report.Time.Milliseconds())
	}
	if s := stdout.String(); s != "solve_test" {
		t.Fatal("Expected:", "solve_test", "got:", s)
	}
}

func TestSafeexecMemoryLimit(t *testing.T) {
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
		Layers:      []string{alpinePath},
		Command:     []string{"/bin/sh", "-c", "echo -n 'solve_test'"},
		TimeLimit:   time.Second,
		MemoryLimit: 1024,
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
	report, err := process.Wait()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if report.ExitCode == 0 {
		t.Fatal("Expected non-zero exit code")
	}
}

func TestSafeexecTimeLimit(t *testing.T) {
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
		Layers:      []string{alpinePath},
		Command:     []string{"/bin/sh", "-c", "sleep 2 && echo -n 'solve_test'"},
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
	report, err := process.Wait()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if report.ExitCode == 0 {
		t.Fatal("Expected non-zero exit code")
	}
	if report.Time < time.Second {
		t.Fatal("Invalid time:", report.Time.Milliseconds())
	}
}

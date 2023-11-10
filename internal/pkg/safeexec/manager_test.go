package safeexec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/udovin/solve/internal/pkg/archives"
)

const (
	testDataDir      = "../../../testdata"
	testSafeexecPath = "../../../cmd/safeexec/safeexec"
	testCgroupName   = "../solve-safeexec"
)

var alpinePath = ""

func TestMain(m *testing.M) {
	os.Exit(func() int {
		tempPath, err := os.MkdirTemp("", "")
		if err != nil {
			panic(fmt.Errorf("cannot prepare alpine rootfs: %w", err))
		}
		defer func() { _ = os.RemoveAll(tempPath) }()
		alpinePath = tempPath
		if err := archives.ExtractTarGz(
			filepath.Join(testDataDir, "alpine-cpp.tar.gz"),
			alpinePath,
		); err != nil {
			panic(fmt.Errorf("cannot extract alpine rootfs: %w", err))
		}
		return m.Run()
	}())
}

func TestSafeexecSimple(t *testing.T) {
	safeexecPath := filepath.Join(t.TempDir(), "safeexec")
	safeexec, err := NewManager(testSafeexecPath, safeexecPath, testCgroupName)
	if err != nil {
		t.Fatal("Error:", err)
	}
	stdout := strings.Builder{}
	processConfig := ProcessConfig{
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
	if report.RealTime < time.Second || report.RealTime > 6*time.Second {
		t.Fatal("Invalid time:", report.RealTime.Milliseconds())
	}
	if s := stdout.String(); s != "solve_test" {
		t.Fatal("Expected:", "solve_test", "got:", s)
	}
}

func TestSafeexecMemoryLimit(t *testing.T) {
	safeexecPath := filepath.Join(t.TempDir(), "safeexec")
	safeexec, err := NewManager(testSafeexecPath, safeexecPath, testCgroupName)
	if err != nil {
		t.Fatal("Error:", err)
	}
	stdout := strings.Builder{}
	processConfig := ProcessConfig{
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
	if report.Memory <= 1024 {
		t.Fatal("Invalid memory:", report.Memory)
	}
}

func TestSafeexecTimeLimit(t *testing.T) {
	safeexecPath := filepath.Join(t.TempDir(), "safeexec")
	safeexec, err := NewManager(testSafeexecPath, safeexecPath, testCgroupName)
	if err != nil {
		t.Fatal("Error:", err)
	}
	stdout := strings.Builder{}
	processConfig := ProcessConfig{
		Layers:      []string{alpinePath},
		Command:     []string{"/bin/sh", "-c", "sleep 3 && echo -n 'solve_test'"},
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
	if report.RealTime < time.Second {
		t.Fatal("Invalid time:", report.RealTime.Milliseconds())
	}
}

func TestSafeexecCancel(t *testing.T) {
	safeexecPath := filepath.Join(t.TempDir(), "safeexec")
	safeexec, err := NewManager(testSafeexecPath, safeexecPath, testCgroupName)
	if err != nil {
		t.Fatal("Error:", err)
	}
	stdout := strings.Builder{}
	processConfig := ProcessConfig{
		Layers:      []string{alpinePath},
		Command:     []string{"/bin/sh", "-c", "sleep 1 && echo -n 'solve_test'"},
		TimeLimit:   2 * time.Second,
		MemoryLimit: 1024 * 1024,
		Stdout:      &stdout,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	process, err := safeexec.Create(ctx, processConfig)
	if err != nil {
		t.Fatal("Error:", err)
	}
	defer func() { _ = process.Release() }()
	if err := process.Start(); err != nil {
		t.Fatal("Error:", err)
	}
	cancel()
	report, err := process.Wait()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if report.ExitCode == 0 {
		t.Fatal("Expected non-zero exit code")
	}
}

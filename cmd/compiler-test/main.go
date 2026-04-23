package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/udovin/solve/internal/models"
	"github.com/udovin/solve/internal/pkg/archives"
	"github.com/udovin/solve/internal/pkg/compilers"
	compilerCache "github.com/udovin/solve/internal/pkg/compilers/cache"
	"github.com/udovin/solve/internal/pkg/safeexec"
)

type compilerConfig struct {
	Name   string                `json:"name"`
	Config models.CompilerConfig `json:"config"`
}

type testCaseConfig struct {
	ExitCode int `json:"exit_code"`
}

func main() {
	safeexecPath := flag.String("safeexec", "", "Path to safeexec binary")
	cgroupName := flag.String("cgroup", "../solve-compiler-test", "Cgroup name")
	compilerDir := flag.String("compiler-dir", "", "Path to compiler directory")
	imagePath := flag.String("image", "", "Path to compiler image tar.gz")
	reset := flag.Bool("reset", false, "Reset canonical files")
	flag.Parse()
	if *safeexecPath == "" || *compilerDir == "" || *imagePath == "" {
		fmt.Fprintln(os.Stderr, "usage: compiler-test --safeexec PATH --compiler-dir DIR --image PATH [--reset]")
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, *safeexecPath, *cgroupName, *compilerDir, *imagePath, *reset); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, safeexecPath, cgroupName, compilerDir, imagePath string, reset bool) error {
	// Load compiler config.
	var config compilerConfig
	configPath := filepath.Join(compilerDir, "config.json")
	if err := decodeJSONFile(configPath, &config); err != nil {
		return fmt.Errorf("cannot read compiler config: %w", err)
	}
	// Extract image to temp dir.
	layerDir, err := os.MkdirTemp("", "compiler-test-layer-*")
	if err != nil {
		return fmt.Errorf("cannot create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(layerDir) }()
	if err := archives.ExtractTarGz(imagePath, layerDir); err != nil {
		return fmt.Errorf("cannot extract image: %w", err)
	}
	// Create safeexec manager.
	executionDir, err := os.MkdirTemp("", "compiler-test-exec-*")
	if err != nil {
		return fmt.Errorf("cannot create execution dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(executionDir) }()
	mgr, err := safeexec.NewManager(safeexecPath, executionDir, cgroupName)
	if err != nil {
		return fmt.Errorf("cannot create safeexec manager: %w", err)
	}
	// Create compiler.
	compiler := compilerCache.NewCompiler(config.Name, layerDir, config.Config, mgr)
	// Find and run test cases.
	testsDir := filepath.Join(compilerDir, "tests")
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		return fmt.Errorf("cannot read tests dir: %w", err)
	}
	passed := 0
	failed := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		caseName := entry.Name()
		caseDir := filepath.Join(testsDir, caseName)
		ok, err := runTestCase(ctx, compiler, config.Config, caseDir, caseName, reset)
		if err != nil {
			return fmt.Errorf("test %s: %w", caseName, err)
		}
		if ok {
			passed++
		} else {
			failed++
		}
	}
	fmt.Printf("\n%s: %d passed, %d failed\n", config.Name, passed, failed)
	if failed > 0 {
		return fmt.Errorf("%d test(s) failed", failed)
	}
	return nil
}

func runTestCase(
	ctx context.Context,
	compiler compilers.Compiler,
	config models.CompilerConfig,
	caseDir, caseName string,
	reset bool,
) (bool, error) {
	// Load test case config.
	var caseConfig testCaseConfig
	if err := decodeJSONFile(filepath.Join(caseDir, "config.json"), &caseConfig); err != nil {
		return false, fmt.Errorf("cannot read case config: %w", err)
	}
	// Find solution file.
	solutionPath, err := findSolution(caseDir, config.Extensions)
	if err != nil {
		return false, err
	}
	// Compile.
	targetDir, err := os.MkdirTemp("", "compiler-test-target-*")
	if err != nil {
		return false, err
	}
	defer func() { _ = os.RemoveAll(targetDir) }()
	targetPath := filepath.Join(targetDir, "output")
	report, err := compiler.Compile(ctx, compilers.CompileOptions{
		Source:      solutionPath,
		Target:      targetPath,
		TimeLimit:   30 * time.Second,
		MemoryLimit: compilers.CompileMemoryLimit,
	})
	if err != nil {
		return false, fmt.Errorf("compile error: %w", err)
	}
	// Serialize diagnostics to NDJSON.
	var diagBuf strings.Builder
	for _, d := range report.Diagnostics {
		data, err := json.Marshal(d)
		if err != nil {
			return false, fmt.Errorf("cannot marshal diagnostic: %w", err)
		}
		diagBuf.Write(data)
		diagBuf.WriteByte('\n')
	}
	if reset {
		// Write canonical files.
		if err := os.WriteFile(filepath.Join(caseDir, "log.txt"), []byte(report.Log), 0644); err != nil {
			return false, err
		}
		if err := os.WriteFile(filepath.Join(caseDir, "diagnostics.ndjson"), []byte(diagBuf.String()), 0644); err != nil {
			return false, err
		}
		fmt.Printf("RESET %s\n", caseName)
		return true, nil
	}
	// Compare against canonical files.
	ok := true
	if report.ExitCode != caseConfig.ExitCode {
		fmt.Printf("FAIL  %s: exit_code: expected %d, got %d\n", caseName, caseConfig.ExitCode, report.ExitCode)
		ok = false
	}
	if !compareFile(filepath.Join(caseDir, "log.txt"), report.Log, caseName, "log.txt") {
		ok = false
	}
	if !compareFile(filepath.Join(caseDir, "diagnostics.ndjson"), diagBuf.String(), caseName, "diagnostics.ndjson") {
		ok = false
	}
	if ok {
		fmt.Printf("PASS  %s\n", caseName)
	}
	return ok, nil
}

func findSolution(caseDir string, extensions []string) (string, error) {
	for _, ext := range extensions {
		path := filepath.Join(caseDir, "solution."+ext)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no solution file found in %s (expected extensions: %v)", caseDir, extensions)
}

func compareFile(path, actual, caseName, fileName string) bool {
	expected, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No canonical file — skip check.
			return true
		}
		fmt.Printf("FAIL  %s: cannot read %s: %v\n", caseName, fileName, err)
		return false
	}
	if string(expected) != actual {
		fmt.Printf("FAIL  %s: %s mismatch\n", caseName, fileName)
		printDiff(string(expected), actual)
		return false
	}
	return true
}

func printDiff(expected, actual string) {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")
	maxLen := len(expectedLines)
	if len(actualLines) > maxLen {
		maxLen = len(actualLines)
	}
	for i := 0; i < maxLen; i++ {
		var exp, act string
		if i < len(expectedLines) {
			exp = expectedLines[i]
		}
		if i < len(actualLines) {
			act = actualLines[i]
		}
		if exp != act {
			fmt.Printf("  line %d:\n", i+1)
			fmt.Printf("    expected: %s\n", exp)
			fmt.Printf("    actual:   %s\n", act)
		}
	}
}

func decodeJSONFile(path string, v any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	return json.NewDecoder(file).Decode(v)
}

package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/udovin/solve/internal/managers"
	"github.com/udovin/solve/internal/models"
)

func TestProblemsSimpleScenario(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	user := NewTestUser(e)
	user.AddRoles("create_problem")
	user.LoginClient()
	file, err := os.Open(filepath.Join(testDataDir, "a-plus-b.zip"))
	if err != nil {
		t.Fatal("Error:", err)
	}
	form := CreateProblemForm{}
	form.Title = getPtr("a-plus-b")
	form.PackageFile = managers.NewFileReader(file)
	problem, err := e.Client.CreateProblem(context.Background(), form)
	if err != nil {
		t.Fatal("Error:", err)
	}
	e.Check(problem)
	{
		file, err := os.Open(filepath.Join(testDataDir, "a-plus-b.zip"))
		if err != nil {
			t.Fatal("Error:", err)
		}
		form := UpdateProblemForm{}
		form.Title = getPtr("a-plus-b-2")
		form.PackageFile = managers.NewFileReader(file)
		updated, err := e.Client.UpdateProblem(context.Background(), problem.ID, form)
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(updated)
	}
	{
		deleted, err := e.Client.DeleteProblem(context.Background(), problem.ID)
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(deleted)
	}
}

func NewTestCompiler(e *TestEnv) Compiler {
	file, err := os.Open(filepath.Join(testDataDir, "alpine-cpp.tar.gz"))
	if err != nil {
		e.tb.Fatal("Error:", err)
	}
	config := models.CompilerConfig{
		Language:   "C++",
		Compiler:   "alpine-cpp",
		Extensions: []string{"cpp"},
		Compile: &models.CompilerCommandConfig{
			Command: "g++ --std=c++17 -O2 -DONLINE_JUDGE -o solution solution.cpp",
			Source:  getPtr("solution.cpp"),
			Binary:  getPtr("solution"),
			Environ: []string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			},
			Workdir: "/home/judge",
		},
		Execute: &models.CompilerCommandConfig{
			Command: "./solution",
			Binary:  getPtr("solution"),
			Workdir: "/home/judge",
		},
	}
	rawConfig, err := json.Marshal(config)
	if err != nil {
		e.tb.Fatal("Error:", err)
	}
	form := CreateCompilerForm{}
	form.Name = getPtr("alpine-cpp")
	form.Config = JSON{rawConfig}
	form.ImageFile = managers.NewFileReader(file)
	compiler, err := e.Client.CreateCompiler(context.Background(), form)
	if err != nil {
		e.tb.Fatal("Error:", err)
	}
	settings := map[string]string{
		"invoker.compilers.polygon.cpp.g++17": "alpine-cpp",
	}
	for key, value := range settings {
		createForm := CreateSettingForm{}
		createForm.Key = getPtr(key)
		createForm.Value = getPtr(value)
		if _, err := e.Client.CreateSetting(context.Background(), createForm); err != nil {
			e.tb.Fatal("Error:", err)
		}
	}
	e.SyncStores()
	return compiler
}

func NewTestProblem(e *TestEnv) Problem {
	file, err := os.Open(filepath.Join(testDataDir, "a-plus-b.zip"))
	if err != nil {
		e.tb.Fatal("Error:", err)
	}
	form := CreateProblemForm{}
	form.Title = getPtr("a-plus-b")
	form.PackageFile = managers.NewFileReader(file)
	problem, err := e.Client.CreateProblem(context.Background(), form)
	if err != nil {
		e.tb.Fatal("Error:", err)
	}
	e.WaitProblemUpdated(problem.ID)
	return problem
}

func NewTestInteractiveProblem(e *TestEnv) Problem {
	file, err := os.Open(filepath.Join(testDataDir, "a-plus-b-interactive.zip"))
	if err != nil {
		e.tb.Fatal("Error:", err)
	}
	form := CreateProblemForm{}
	form.Title = getPtr("a-plus-b-interactive")
	form.PackageFile = managers.NewFileReader(file)
	problem, err := e.Client.CreateProblem(context.Background(), form)
	if err != nil {
		e.tb.Fatal("Error:", err)
	}
	e.WaitProblemUpdated(problem.ID)
	return problem
}

func TestProblemBuildScenario(t *testing.T) {
	e := NewTestEnv(t, WithInvoker{})
	defer e.Close()
	user := NewTestUser(e)
	user.AddRoles("create_compiler", "create_problem", "create_setting")
	user.LoginClient()
	// Create compiler.
	NewTestCompiler(e)
	// Create problem.
	NewTestProblem(e)
	// Create interactive problem.
	NewTestInteractiveProblem(e)
}

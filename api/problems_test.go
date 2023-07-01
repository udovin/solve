package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/udovin/solve/managers"
	"github.com/udovin/solve/models"
)

func TestProblemsSimpleScenario(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	user := NewTestUser(e)
	user.AddRoles(models.CreateProblemRole)
	user.LoginClient()
	file, err := os.Open(filepath.Join("../testdata", "a-plus-b.zip"))
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
		file, err := os.Open(filepath.Join("../testdata", "a-plus-b.zip"))
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

func TestProblemBuildScenario(t *testing.T) {
	e := NewTestEnv(t, WithInvoker{})
	defer e.Close()
	user := NewTestUser(e)
	user.AddRoles(
		models.CreateContestRole,
		models.CreateCompilerRole,
		models.CreateProblemRole,
		models.CreateSettingRole,
	)
	user.LoginClient()
	// Create compiler.
	{
		file, err := os.Open(filepath.Join("../testdata", "alpine-cpp.tar.gz"))
		if err != nil {
			t.Fatal("Error:", err)
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
			t.Fatal("Error:", err)
		}
		form := CreateCompilerForm{}
		form.Name = getPtr("alpine-cpp")
		form.Config = JSON{rawConfig}
		form.ImageFile = managers.NewFileReader(file)
		if _, err := e.Client.CreateCompiler(context.Background(), form); err != nil {
			t.Fatal("Error:", err)
		}
	}
	// Setup polygon compilers.
	{
		settings := map[string]string{
			"invoker.compilers.polygon.cpp.g++17": "alpine-cpp",
		}
		for key, value := range settings {
			createForm := CreateSettingForm{}
			createForm.Key = getPtr(key)
			createForm.Value = getPtr(value)
			if _, err := e.Client.CreateSetting(context.Background(), createForm); err != nil {
				t.Fatal("Error:", err)
			}
		}
		e.SyncStores()
	}
	// Create problem
	{
		file, err := os.Open(filepath.Join("../testdata", "a-plus-b.zip"))
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
		e.WaitProblemUpdated(problem.ID)
	}
}

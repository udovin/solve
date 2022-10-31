package api

import (
	"context"
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
	e.SyncStores()
	{
		deleted, err := e.Client.DeleteProblem(context.Background(), problem.ID)
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(deleted)
	}
}

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

func TestCompilersSimpleScenario(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	user := NewTestUser(e)
	user.AddRoles(
		models.CreateCompilerRole,
		models.UpdateCompilerRole,
		models.DeleteCompilerRole,
	)
	user.LoginClient()
	file, err := os.Open(filepath.Join("../testdata", "alpine.tar.gz"))
	if err != nil {
		t.Fatal("Error:", err)
	}
	config := models.CompilerConfig{
		Extensions: []string{"cpp", "h", "hpp", "c++", "cxx"},
	}
	rawConfig, err := json.Marshal(config)
	if err != nil {
		t.Fatal("Error:", err)
	}
	form := CreateCompilerForm{}
	form.Name = getPtr("test")
	form.Config = JSON{rawConfig}
	form.ImageFile = managers.NewFileReader(file)
	compiler, err := e.Client.CreateCompiler(context.Background(), form)
	if err != nil {
		t.Fatal("Error:", err)
	}
	e.Check(compiler)
	{
		file, err := os.Open(filepath.Join("../testdata", "alpine.tar.gz"))
		if err != nil {
			t.Fatal("Error:", err)
		}
		form := UpdateCompilerForm{}
		form.Name = getPtr("test2")
		form.ImageFile = managers.NewFileReader(file)
		updated, err := e.Client.UpdateCompiler(context.Background(), compiler.ID, form)
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(updated)
	}
	e.SyncStores()
	{
		deleted, err := e.Client.DeleteCompiler(context.Background(), compiler.ID)
		if err != nil {
			t.Fatal("Error:", err)
		}
		e.Check(deleted)
	}
}

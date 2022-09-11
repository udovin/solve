package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/udovin/solve/models"
)

func TestCompilersSimpleScenario(t *testing.T) {
	e := NewTestEnv(t)
	defer e.Close()
	user := NewTestUser(e)
	user.AddRoles(models.CreateCompilerRole)
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
	compiler, err := e.Client.CreateCompiler(context.Background(), CreateCompilerForm{
		Name:      "test",
		Config:    JSON{rawConfig},
		ImageFile: file,
	})
	if err != nil {
		t.Fatal("Error:", err)
	}
	e.Check(compiler)
}

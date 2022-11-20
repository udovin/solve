package invoker

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/udovin/solve/pkg"
)

func TestProcessor(t *testing.T) {
	containersPath := filepath.Join(t.TempDir(), "containers")
	alpinePath := filepath.Join(t.TempDir(), "alpine")
	if err := pkg.ExtractTarGz(
		filepath.Join("../testdata", "alpine.tar.gz"),
		alpinePath,
	); err != nil {
		t.Fatal("Error:", err)
	}
	factory, err := newFactory(containersPath)
	if err != nil {
		t.Fatal("Error:", err)
	}
	stdout := strings.Builder{}
	containerConfig := containerConfig{
		Layers: []string{alpinePath},
		Init: processConfig{
			Args:   []string{"/bin/sh", "-c", "echo -n 'solve_test'"},
			Stdout: &stdout,
		},
	}
	container, err := factory.Create(containerConfig)
	if err != nil {
		t.Fatal("Error:", err)
	}
	defer func() {
		if err := container.Destroy(); err != nil {
			t.Fatal("Error:", err)
		}
	}()
	{
		process, err := container.Start()
		if err != nil {
			t.Fatal("Error:", err)
		}
		if _, err := process.Wait(); err != nil {
			t.Fatal("Error:", err)
		}
		if s := stdout.String(); s != "solve_test" {
			t.Fatal("Expected:", "solve_test", "got:", s)
		}
	}
}

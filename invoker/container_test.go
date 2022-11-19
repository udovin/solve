package invoker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runc/libcontainer"
	"github.com/udovin/solve/pkg"
)

func TestProcessor(t *testing.T) {
	libcontainerPath := filepath.Join(t.TempDir(), "libcontainer")
	processorPath := filepath.Join(t.TempDir(), "processor")
	alpinePath := filepath.Join(t.TempDir(), "alpine")
	if err := pkg.ExtractTarGz(
		filepath.Join("../testdata", "alpine.tar.gz"),
		alpinePath,
	); err != nil {
		t.Fatal("Error:", err)
	}
	factory, err := libcontainer.New(
		libcontainerPath,
		libcontainer.InitArgs(os.Args[0], "init"),
	)
	processor := Processor{
		factory: factory,
		dir:     processorPath,
	}
	containerConfig := ContainerConfig{
		Layers: []string{alpinePath},
		Init: ProcessConfig{
			Args: []string{"/bin/sh", "-c", "echo 'Hello, World'"},
		},
	}
	container, err := processor.Create(containerConfig)
	if err != nil {
		t.Fatal("Error:", err)
	}
	defer func() {
		if err := container.Destroy(); err != nil {
			t.Fatal("Error:", err)
		}
	}()
	process, err := container.Start()
	if err != nil {
		t.Fatal("Error:", err)
	}
	if _, err := process.Wait(); err != nil {
		t.Fatal("Error:", err)
	}
}

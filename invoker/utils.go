package invoker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gofrs/uuid"
)

func makeTempDir() (string, error) {
	for i := 0; i < 100; i++ {
		name, err := uuid.NewV4()
		if err != nil {
			return "", err
		}
		dirPath := filepath.Join(os.TempDir(), name.String())
		if err := os.MkdirAll(dirPath, 0777); err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", err
		}
		return dirPath, nil
	}
	return "", fmt.Errorf("unable to create temp directory")
}

func copyFileRec(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
		return err
	}
	return copyFile(source, target)
}

func copyFile(source, target string) error {
	r, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		_ = r.Close()
	}()
	w, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() {
		_ = w.Close()
	}()
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return nil
}

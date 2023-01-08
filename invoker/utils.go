package invoker

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func makeTempDir() (string, error) {
	for i := 0; i < 100; i++ {
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			return "", err
		}
		dirPath := filepath.Join(os.TempDir(), hex.EncodeToString(bytes))
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
	defer func() { _ = r.Close() }()
	stat, err := r.Stat()
	if err != nil {
		return err
	}
	w, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() { _ = w.Close() }()
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return os.Chmod(w.Name(), stat.Mode())
}

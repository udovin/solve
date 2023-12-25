package utils

import (
	"io"
	"os"
	"path/filepath"
)

func CopyFileRec(target, source string) error {
	if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
		return err
	}
	return CopyFile(target, source)
}

func CopyFile(target, source string) error {
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

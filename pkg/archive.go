package pkg

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

func ExtractZip(source, target string) error {
	archive, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer func() {
		_ = archive.Close()
	}()
	for _, file := range archive.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, os.ModePerm); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
			return err
		}
		if err := func() error {
			input, err := file.Open()
			if err != nil {
				return err
			}
			defer func() {
				_ = input.Close()
			}()
			output, err := os.OpenFile(
				path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode(),
			)
			if err != nil {
				return err
			}
			defer func() {
				_ = output.Close()
			}()
			_, err = io.Copy(output, input)
			return err
		}(); err != nil {
			return err
		}
	}
	return nil
}

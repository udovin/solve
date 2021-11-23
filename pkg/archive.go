package pkg

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
)

// ExtractTarGz extracts tar.gz archive into specified path.
func ExtractTarGz(source, target string) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer func() {
		_ = reader.Close()
	}()
	if err := os.MkdirAll(target, os.ModePerm); err != nil {
		return err
	}
	archive := tar.NewReader(reader)
	for {
		header, err := archive.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if header == nil {
			continue
		}
		path := filepath.Join(target, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(
				path, os.FileMode(header.Mode),
			); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := func() error {
				output, err := os.OpenFile(
					path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
					os.FileMode(header.Mode),
				)
				if err != nil {
					return err
				}
				defer func() {
					_ = output.Close()
				}()
				_, err = io.Copy(output, archive)
				return err
			}(); err != nil {
				return err
			}
		}
	}
}

// ExtractZip extracts zip archive into specified path.
func ExtractZip(source, target string) error {
	archive, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer func() {
		_ = archive.Close()
	}()
	if err := os.MkdirAll(target, os.ModePerm); err != nil {
		return err
	}
	for _, file := range archive.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			if err := os.Mkdir(path, file.Mode()); err != nil {
				return err
			}
			continue
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

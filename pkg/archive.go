package pkg

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractZip extracts zip archive into specified path.
func ExtractZip(source, target string) (errRes error) {
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
	defer func() {
		if errRes != nil {
			_ = os.RemoveAll(target)
		}
	}()
	for _, file := range archive.File {
		if strings.Contains(file.Name, "..") {
			return fmt.Errorf("illegal file path: %q", file.Name)
		}
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
			if _, err := io.Copy(output, input); err != nil {
				return err
			}
			return output.Sync()
		}(); err != nil {
			return err
		}
	}
	return nil
}

// ExtractTarGz extracts tar.gz archive into specified path.
func ExtractTarGz(source, target string) (errRes error) {
	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("cannot open gzip reader: %w", err)
	}
	defer func() {
		_ = reader.Close()
	}()
	if err := os.MkdirAll(target, os.ModePerm); err != nil {
		return fmt.Errorf("cannot prepare target directory: %w", err)
	}
	defer func() {
		if errRes != nil {
			_ = os.RemoveAll(target)
		}
	}()
	archive := tar.NewReader(reader)
	links := map[string]string{}
	symlinks := map[string]string{}
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("cannot get header: %w", err)
		}
		if header == nil {
			continue
		}
		if strings.Contains(header.Name, "..") {
			return fmt.Errorf("illegal file path: %q", header.Name)
		}
		path := filepath.Join(target, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(
				path, os.FileMode(header.Mode),
			); err != nil {
				return fmt.Errorf("cannot create directory %q: %w", header.Name, err)
			}
		case tar.TypeReg:
			if err := func() error {
				output, err := os.OpenFile(
					path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
					os.FileMode(header.Mode),
				)
				if err != nil {
					return fmt.Errorf("cannot create file %q: %w", header.Name, err)
				}
				defer func() {
					_ = output.Close()
				}()
				if _, err := io.Copy(output, archive); err != nil {
					return fmt.Errorf("cannot write file %q: %w", header.Name, err)
				}
				return output.Sync()
			}(); err != nil {
				return err
			}
		case tar.TypeLink:
			links[path] = filepath.Join(target, header.Linkname)
		case tar.TypeSymlink:
			symlinks[path] = header.Linkname
		default:
			return fmt.Errorf("unsupported type %q in %q", header.Typeflag, header.Name)
		}
	}
	for path, link := range links {
		if err := os.Link(link, path); err != nil {
			return err
		}
	}
	for path, link := range symlinks {
		if err := os.Symlink(link, path); err != nil {
			return err
		}
	}
	return nil
}

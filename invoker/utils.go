package invoker

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func readFile(name string, limit int) (string, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", err
	}
	bytes := make([]byte, limit+1)
	read, err := file.Read(bytes)
	if err != nil && err != io.EOF {
		return "", err
	}
	if read > limit {
		return fixUTF8String(string(bytes[:limit])) + "...", nil
	}
	return fixUTF8String(string(bytes[:read])), nil
}

func fixUTF8String(s string) string {
	return strings.ReplaceAll(strings.ToValidUTF8(s, ""), "\u0000", "")
}

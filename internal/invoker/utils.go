package invoker

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

type truncateBuffer struct {
	buffer strings.Builder
	limit  int
	mutex  sync.Mutex
}

func (b *truncateBuffer) String() string {
	return fixUTF8String(b.buffer.String())
}

func (b *truncateBuffer) Write(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	l := len(p)
	if b.buffer.Len()+l > b.limit {
		p = p[:b.limit-b.buffer.Len()]
	}
	if len(p) == 0 {
		return l, nil
	}
	n, err := b.buffer.Write(p)
	if err != nil {
		return n, err
	}
	return l, nil
}

func fixUTF8String(s string) string {
	return strings.ReplaceAll(strings.ToValidUTF8(s, ""), "\u0000", "")
}

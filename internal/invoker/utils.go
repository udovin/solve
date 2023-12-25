package invoker

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

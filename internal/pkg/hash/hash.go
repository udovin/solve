package hash

import (
	"crypto/md5"
	"encoding/hex"
	"io"
)

func CalculateMD5(r io.Reader) (string, int64, error) {
	hash := md5.New()
	size, err := io.Copy(hash, r)
	if err != nil {
		return "", size, err
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

func HashFileToSHA256(r io.Reader) (hashHex string, size int64, err error) {
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

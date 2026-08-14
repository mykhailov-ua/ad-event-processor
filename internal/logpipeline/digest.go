package logpipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

type fileDigest struct {
	SHA256 string
	Size   int64
}

func computeFileDigest(path string) (fileDigest, error) {
	file, err := os.Open(path)
	if err != nil {
		return fileDigest{}, err
	}
	defer func() { _ = file.Close() }()

	sha := sha256.New()
	written, err := io.Copy(sha, file)
	if err != nil {
		return fileDigest{}, err
	}

	return fileDigest{
		SHA256: hex.EncodeToString(sha.Sum(nil)),
		Size:   written,
	}, nil
}

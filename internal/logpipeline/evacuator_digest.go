package logpipeline

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

type fileDigests struct {
	SHA256 string
	MD5    string
	Size   int64
}

func computeFileDigests(path string) (fileDigests, error) {
	file, err := os.Open(path)
	if err != nil {
		return fileDigests{}, err
	}
	defer func() { _ = file.Close() }()

	sha := sha256.New()
	md5Hash := md5.New()
	reader := io.TeeReader(file, md5Hash)

	written, err := io.CopyBuffer(sha, reader, copyBuffer())
	if err != nil {
		return fileDigests{}, err
	}

	return fileDigests{
		SHA256: hex.EncodeToString(sha.Sum(nil)),
		MD5:    fmt.Sprintf("%x", md5Hash.Sum(nil)),
		Size:   written,
	}, nil
}

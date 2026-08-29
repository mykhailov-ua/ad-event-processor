package verify

import (
	"os"
	"strings"
)

func ReadFileTrim(path string) ([]byte, error) {
	return readFileTrim(path)
}

func readFileTrim(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, ErrInvalidTokenFormat
	}
	return []byte(trimmed), nil
}

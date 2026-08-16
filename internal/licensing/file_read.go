package licensing

import (
	"os"
	"strings"
)

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

package h2frametrace

import (
	"hash/crc32"
	"strings"
)

const MaxTokenLen = 64

func NormalizeTokens(raw string) (string, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return "", errEmpty
	}
	if len(raw) > 512 {
		return "", errTooLong
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if len(part) > MaxTokenLen {
			return "", errTokenTooLong
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return "", errEmpty
	}
	return strings.Join(out, ","), nil
}

func HashTokens(normalized string) uint32 {
	if normalized == "" {
		return 0
	}
	return crc32.ChecksumIEEE([]byte(normalized))
}

func HashFromWire(raw string) (uint32, string, error) {
	norm, err := NormalizeTokens(raw)
	if err != nil {
		return 0, "", err
	}
	return HashTokens(norm), norm, nil
}

var (
	errEmpty        = validationError("empty h2 frame trace")
	errTooLong      = validationError("h2 frame trace too long")
	errTokenTooLong = validationError("h2 frame trace token too long")
)

type validationError string

func (e validationError) Error() string { return string(e) }

// Package tcpsynopt hashes TCP SYN option-order fingerprints for corpus matching.
//
// Wire format (X-TCP-SIG-V2 header value): comma-separated option tokens from the SYN
// option walk, e.g. "nop,nop,mss:1460,sackok,ts". Tokens are lower-case ASCII.
// Hash is CRC32-IEEE over the normalized token string (same family as JA3 corpus keys).
//
// Edge/XDP emits the token string after TLV-walking TCP options; tracker compares the
// hash against an embedded corpus (fail-open when header absent or corpus row missing).
//
// Verify:
// go test ./pkg/tcpsynopt/ -count=1
// go test ./internal/ingest/ -short -run TCPSynOpt -count=1
package tcpsynopt

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
	errEmpty        = validationError("empty tcp syn option trace")
	errTooLong      = validationError("tcp syn option trace too long")
	errTokenTooLong = validationError("tcp syn option token too long")
)

type validationError string

func (e validationError) Error() string { return string(e) }

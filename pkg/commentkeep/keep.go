package commentkeep

import (
	"strings"
	"unicode"
)

func Keep(text string) bool {
	t := strings.TrimSpace(text)
	if t == "" {
		return false
	}
	if strings.HasPrefix(t, "go:") || strings.HasPrefix(t, "nolint:") {
		return true
	}
	if IsDivider(t) {
		return false
	}
	return IsTechnical(t)
}

func IsDivider(text string) bool {
	t := strings.TrimSpace(text)
	if len(t) < 3 {
		return false
	}
	r := rune(t[0])
	if r != '-' && r != '=' && r != '_' && r != '*' && r != '#' && r != '─' && r != '━' && r != '═' {
		return false
	}
	for _, c := range t {
		if c != r && !unicode.IsSpace(c) {
			return false
		}
	}
	return true
}

func IsTechnical(text string) bool {
	lower := strings.ToLower(text)
	for _, m := range technicalMarkers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

var technicalMarkers = []string{
	"unsafe",
	"keepalive",
	"bce",
	"panicindex",
	"escape",
	"mmap",
	"fsync",
	"atomic",
	"inline",
	"noinline",
	"vtproto",
	"gnet",
	"race",
	"lock-free",
	"lockfree",
	"alloc",
	"heap",
	"nogc",
	"cache line",
	"cacheline",
	"sync.",
	"mutex",
	"goroutine",
	"lifetime",
	"zero-copy",
	"bounds check",
	"must not",
	"must hold",
	"caller must",
	"do not ",
	"never ",
	"sha256",
	"hmac",
	"tls",
	"x509",
	"endian",
	"syscall",
	"ioctl",
	"bpf",
	"xdp",
	"redis",
	"lua",
	"crc32",
	"protobuf",
	"sqlc",
	"pgx",
	"clickhouse",
	"outbox",
	"idempot",
	"quorum",
	"fencing",
	"semver",
	"cgroup",
	"ringbuf",
	"uprobe",
	"kprobe",
	"perf_event",
}

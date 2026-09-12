package moderatorcorpus

import (
	"errors"
	"hash/crc32"
	"strconv"
	"strings"
)

const MaxFieldLen = 512

type Tuple struct {
	JA3              string
	JA4              string
	TCPSig           string
	WebGLRenderer    string
	LayerDesyncCount uint8
}

type Entry struct {
	JA3Hash     uint32
	JA4Hash     uint32
	TCPHash     uint32
	WebGLHash   uint32
	DesyncMin   uint8
	RawJA3      string
	RawJA4      string
	RawTCPSig   string
	RawWebGL    string
	DesyncCount uint8
}

type Snapshot struct {
	Gen   uint64
	ByJA3 map[uint32][]Entry
}

func FieldHash(b []byte) uint32 {
	if len(b) == 0 {
		return 0
	}
	return crc32.ChecksumIEEE(b)
}

func ParseTCPSigHex(raw string) (uint32, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		raw = raw[2:]
	}
	if len(raw) > 8 {
		return 0, false
	}
	var val uint32
	for i := range len(raw) {
		c := raw[i]
		var digit uint32
		switch {
		case c >= '0' && c <= '9':
			digit = uint32(c - '0')
		case c >= 'a' && c <= 'f':
			digit = uint32(c-'a') + 10
		case c >= 'A' && c <= 'F':
			digit = uint32(c-'A') + 10
		default:
			return 0, false
		}
		val = (val << 4) | digit
	}
	return val, true
}

func NormalizeTuple(t Tuple) (Tuple, error) {
	t.JA3 = strings.TrimSpace(t.JA3)
	t.JA4 = strings.TrimSpace(t.JA4)
	t.TCPSig = strings.TrimSpace(t.TCPSig)
	t.WebGLRenderer = strings.TrimSpace(t.WebGLRenderer)
	if t.JA3 == "" {
		return Tuple{}, errJA3Required
	}
	if len(t.JA3) > MaxFieldLen || len(t.JA4) > MaxFieldLen || len(t.TCPSig) > MaxFieldLen || len(t.WebGLRenderer) > MaxFieldLen {
		return Tuple{}, errFieldTooLong
	}
	if t.TCPSig != "" {
		if _, ok := ParseTCPSigHex(t.TCPSig); !ok {
			return Tuple{}, errInvalidTCPSig
		}
	}
	return t, nil
}

func EntryFromTuple(t Tuple) (Entry, error) {
	norm, err := NormalizeTuple(t)
	if err != nil {
		return Entry{}, err
	}
	entry := Entry{
		JA3Hash:     FieldHash([]byte(norm.JA3)),
		RawJA3:      norm.JA3,
		RawJA4:      norm.JA4,
		RawTCPSig:   norm.TCPSig,
		RawWebGL:    norm.WebGLRenderer,
		DesyncCount: norm.LayerDesyncCount,
		DesyncMin:   norm.LayerDesyncCount,
	}
	if norm.JA4 != "" {
		entry.JA4Hash = FieldHash([]byte(norm.JA4))
	}
	if norm.TCPSig != "" {
		if tcp, ok := ParseTCPSigHex(norm.TCPSig); ok {
			entry.TCPHash = tcp
		}
	}
	if norm.WebGLRenderer != "" {
		entry.WebGLHash = FieldHash([]byte(norm.WebGLRenderer))
	}
	return entry, nil
}

func BuildSnapshot(entries []Entry, gen uint64) *Snapshot {
	if len(entries) == 0 {
		return &Snapshot{Gen: gen, ByJA3: map[uint32][]Entry{}}
	}
	byJA3 := make(map[uint32][]Entry, len(entries))
	for _, e := range entries {
		if e.JA3Hash == 0 {
			continue
		}
		byJA3[e.JA3Hash] = append(byJA3[e.JA3Hash], e)
	}
	return &Snapshot{Gen: gen, ByJA3: byJA3}
}

func MatchSnapshot(snap *Snapshot, ja3, ja4 []byte, tcpSig uint32, tcpSigSet uint8, webgl []byte, layerDesync uint8) bool {
	if snap == nil || len(snap.ByJA3) == 0 || len(ja3) == 0 {
		return false
	}
	cands := snap.ByJA3[FieldHash(ja3)]
	if len(cands) == 0 {
		return false
	}
	var ja4Hash uint32
	if len(ja4) > 0 {
		ja4Hash = FieldHash(ja4)
	}
	var webglHash uint32
	if len(webgl) > 0 {
		webglHash = FieldHash(webgl)
	}
	for _, e := range cands {
		if e.JA4Hash != 0 && e.JA4Hash != ja4Hash {
			continue
		}
		if e.TCPHash != 0 {
			if tcpSigSet == 0 || e.TCPHash != tcpSig {
				continue
			}
		}
		if e.WebGLHash != 0 && e.WebGLHash != webglHash {
			continue
		}
		if e.DesyncMin != 0 && layerDesync < e.DesyncMin {
			continue
		}
		return true
	}
	return false
}

func ParseFeed(data []byte) ([]Entry, error) {
	if len(data) == 0 {
		return nil, nil
	}
	lines := strings.Split(string(data), "\n")
	out := make([]Entry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tuple, err := ParseFeedLine(line)
		if err != nil {
			return nil, err
		}
		entry, err := EntryFromTuple(tuple)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, nil
}

func ParseFeedLine(line string) (Tuple, error) {
	t := Tuple{}
	parts := strings.Split(line, "|")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, ok := strings.Cut(part, ":")
		if !ok {
			return Tuple{}, errInvalidFeedLine
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "ja3":
			t.JA3 = val
		case "ja4":
			t.JA4 = val
		case "tcp":
			t.TCPSig = val
		case "webgl":
			t.WebGLRenderer = val
		case "desync":
			n, err := strconv.Atoi(val)
			if err != nil || n < 0 || n > 255 {
				return Tuple{}, errInvalidDesync
			}
			t.LayerDesyncCount = uint8(n)
		default:
			return Tuple{}, errInvalidFeedLine
		}
	}
	return NormalizeTuple(t)
}

func FormatFeedLine(t Tuple) string {
	norm, err := NormalizeTuple(t)
	if err != nil {
		return ""
	}
	parts := []string{"ja3:" + norm.JA3}
	if norm.JA4 != "" {
		parts = append(parts, "ja4:"+norm.JA4)
	}
	if norm.TCPSig != "" {
		parts = append(parts, "tcp:"+norm.TCPSig)
	}
	if norm.WebGLRenderer != "" {
		parts = append(parts, "webgl:"+norm.WebGLRenderer)
	}
	if norm.LayerDesyncCount != 0 {
		parts = append(parts, "desync:"+strconv.Itoa(int(norm.LayerDesyncCount)))
	}
	return strings.Join(parts, "|")
}

func FormatFeed(entries []Entry) []byte {
	if len(entries) == 0 {
		return []byte("# moderator corpus v1\n")
	}
	var b strings.Builder
	b.WriteString("# moderator corpus v1\n")
	for _, e := range entries {
		line := FormatFeedLine(Tuple{
			JA3:              e.RawJA3,
			JA4:              e.RawJA4,
			TCPSig:           e.RawTCPSig,
			WebGLRenderer:    e.RawWebGL,
			LayerDesyncCount: e.DesyncCount,
		})
		if line == "" {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

var (
	errJA3Required     = validationError("ja3 is required")
	errFieldTooLong    = validationError("field exceeds max length")
	errInvalidTCPSig   = validationError("invalid tcp_sig hex")
	errInvalidFeedLine = validationError("invalid feed line")
	errInvalidDesync   = validationError("invalid desync count")
)

type validationError string

func (e validationError) Error() string { return string(e) }

func ValidationError(msg string) error { return validationError(msg) }

func IsValidationError(err error) bool {
	var ve validationError
	return errors.As(err, &ve)
}

package filter

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"ad-event-processor/pkg/h2frametrace"
)

//go:embed h2_frame_trace_corpus_embed.txt
var h2FrameTraceCorpusEmbed []byte

type H2FrameTraceCorpusSnapshot struct {
	hashFamilies map[uint32]uint8
}

var h2FrameTraceCorpusActive atomic.Pointer[H2FrameTraceCorpusSnapshot]

func init() {
	if snap := parseH2FrameTraceCorpus(h2FrameTraceCorpusEmbed); snap != nil {
		h2FrameTraceCorpusActive.Store(snap)
	}
}

func PublishH2FrameTraceCorpus(snap *H2FrameTraceCorpusSnapshot) {
	if snap == nil || len(snap.hashFamilies) == 0 {
		return
	}
	h2FrameTraceCorpusActive.Store(snap)
}

func LoadH2FrameTraceCorpusFromDir(dir string) *H2FrameTraceCorpusSnapshot {
	base := parseH2FrameTraceCorpus(h2FrameTraceCorpusEmbed)
	if dir == "" {
		return base
	}
	data, err := os.ReadFile(filepath.Join(dir, "h2_frame_trace_corpus.txt"))
	if err != nil || len(data) == 0 {
		return base
	}
	overlay := parseH2FrameTraceCorpus(data)
	if overlay == nil {
		return base
	}
	if base == nil || len(base.hashFamilies) == 0 {
		return overlay
	}
	merged := &H2FrameTraceCorpusSnapshot{
		hashFamilies: make(map[uint32]uint8, len(base.hashFamilies)+len(overlay.hashFamilies)),
	}
	for k, v := range base.hashFamilies {
		merged.hashFamilies[k] = v
	}
	for k, v := range overlay.hashFamilies {
		merged.hashFamilies[k] = v
	}
	return merged
}

func parseH2FrameTraceCorpus(data []byte) *H2FrameTraceCorpusSnapshot {
	if len(data) == 0 {
		return nil
	}
	entries := make(map[uint32]uint8)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		hash, ok := parseTCPSigHex([]byte(line[:eq]))
		if !ok {
			continue
		}
		mask := parseSynSigFamilyMask(line[eq+1:])
		if mask == 0 {
			continue
		}
		entries[hash] = mask
	}
	if len(entries) == 0 {
		return nil
	}
	return &H2FrameTraceCorpusSnapshot{hashFamilies: entries}
}

func ParseH2FrameTraceHeader(b []byte) (uint32, bool) {
	hash, _, err := h2frametrace.HashFromWire(UnsafeString(b))
	if err != nil || hash == 0 {
		return 0, false
	}
	return hash, true
}

func H2FrameTraceCorpusMismatch(ua string, traceHash uint32) bool {
	if traceHash == 0 || ua == "" {
		return false
	}
	snap := h2FrameTraceCorpusActive.Load()
	if snap == nil {
		return false
	}
	allowed, ok := snap.hashFamilies[traceHash]
	if !ok || allowed == 0 {
		return false
	}
	family := ScanUAFamily(ua)
	mask := uaFamilySynSigMask(family)
	if mask == 0 {
		return false
	}
	return allowed&mask == 0
}

package filter

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"ad-event-processor/pkg/tcpsynopt"
)

//go:embed tcp_syn_opt_corpus_embed.txt
var tcpSynOptCorpusEmbed []byte

type TCPSynOptCorpusSnapshot struct {
	hashFamilies map[uint32]uint8
}

var tcpSynOptCorpusActive atomic.Pointer[TCPSynOptCorpusSnapshot]

func init() {
	if snap := parseTCPSynOptCorpus(tcpSynOptCorpusEmbed); snap != nil {
		tcpSynOptCorpusActive.Store(snap)
	}
}

func PublishTCPSynOptCorpus(snap *TCPSynOptCorpusSnapshot) {
	if snap == nil || len(snap.hashFamilies) == 0 {
		return
	}
	tcpSynOptCorpusActive.Store(snap)
}

func LoadTCPSynOptCorpusFromDir(dir string) *TCPSynOptCorpusSnapshot {
	base := parseTCPSynOptCorpus(tcpSynOptCorpusEmbed)
	if dir == "" {
		return base
	}
	data, err := os.ReadFile(filepath.Join(dir, "tcp_syn_opt_corpus.txt"))
	if err != nil || len(data) == 0 {
		return base
	}
	overlay := parseTCPSynOptCorpus(data)
	if overlay == nil {
		return base
	}
	if base == nil || len(base.hashFamilies) == 0 {
		return overlay
	}
	merged := &TCPSynOptCorpusSnapshot{
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

func parseTCPSynOptCorpus(data []byte) *TCPSynOptCorpusSnapshot {
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
	return &TCPSynOptCorpusSnapshot{hashFamilies: entries}
}

func ParseTCPSynOptHeader(b []byte) (uint32, bool) {
	hash, _, err := tcpsynopt.HashFromWire(UnsafeString(b))
	if err != nil || hash == 0 {
		return 0, false
	}
	return hash, true
}

func TCPSynOptCorpusMismatch(ua string, optHash uint32) bool {
	if optHash == 0 || ua == "" {
		return false
	}
	snap := tcpSynOptCorpusActive.Load()
	if snap == nil {
		return false
	}
	allowed, ok := snap.hashFamilies[optHash]
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

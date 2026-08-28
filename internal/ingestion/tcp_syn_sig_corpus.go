package ingestion

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

//go:embed tcp_syn_sig_corpus_embed.txt
var tcpSynSigCorpusEmbed []byte

type tcpSynSigCorpusSnapshot struct {
	hashFamilies map[uint32]uint8
}

var tcpSynSigCorpusActive atomic.Pointer[tcpSynSigCorpusSnapshot]

func init() {
	if snap := parseTCPSynSigCorpus(tcpSynSigCorpusEmbed); snap != nil {
		tcpSynSigCorpusActive.Store(snap)
	}
}

func PublishTCPSynSigCorpus(snap *tcpSynSigCorpusSnapshot) {
	if snap == nil || len(snap.hashFamilies) == 0 {
		return
	}
	tcpSynSigCorpusActive.Store(snap)
}

func loadTCPSynSigCorpusFromDir(dir string) *tcpSynSigCorpusSnapshot {
	base := parseTCPSynSigCorpus(tcpSynSigCorpusEmbed)
	if dir == "" {
		return base
	}
	data, err := os.ReadFile(filepath.Join(dir, "tcp_syn_sig_corpus.txt"))
	if err != nil || len(data) == 0 {
		return base
	}
	overlay := parseTCPSynSigCorpus(data)
	if overlay == nil {
		return base
	}
	if base == nil || len(base.hashFamilies) == 0 {
		return overlay
	}
	merged := &tcpSynSigCorpusSnapshot{
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

func parseTCPSynSigCorpus(data []byte) *tcpSynSigCorpusSnapshot {
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
	return &tcpSynSigCorpusSnapshot{hashFamilies: entries}
}

func parseSynSigFamilyMask(raw string) uint8 {
	var mask uint8
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		switch part {
		case "windows":
			mask |= synSigUAWindows
		case "mac":
			mask |= synSigUAMac
		case "linux":
			mask |= synSigUALinux
		case "mobile":
			mask |= synSigUAMobile
		}
	}
	return mask
}

const (
	synSigUAWindows uint8 = 1 << 0
	synSigUAMac     uint8 = 1 << 1
	synSigUALinux   uint8 = 1 << 2
	synSigUAMobile  uint8 = 1 << 3
)

func uaFamilySynSigMask(family uint8) uint8 {
	switch family {
	case uaFamilyWindows:
		return synSigUAWindows
	case uaFamilyMac:
		return synSigUAMac
	case uaFamilyLinux:
		return synSigUALinux
	case uaFamilyMobile:
		return synSigUAMobile
	default:
		return 0
	}
}

func hashTCPSynFields(ttl uint8, window uint16, mss uint8, doff uint8) uint32 {
	h := uint32(ttl)
	h = (h << 5) ^ uint32(window)
	h = (h << 5) ^ uint32(mss)
	h = (h << 3) ^ uint32(doff)
	return h
}

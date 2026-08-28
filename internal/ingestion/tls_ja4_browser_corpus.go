package ingestion

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

//go:embed ja4_browser_corpus_embed.txt
var ja4BrowserCorpusEmbed []byte

const (
	tlsBrowserChrome  uint8 = 1 << 0
	tlsBrowserFirefox uint8 = 1 << 1
	tlsBrowserSafari  uint8 = 1 << 2
	tlsBrowserOkhttp  uint8 = 1 << 3
	tlsBrowserGo      uint8 = 1 << 4
)

type ja4BrowserCorpusSnapshot struct {
	prefixFamilies map[string]uint8
}

var ja4BrowserCorpusActive atomic.Pointer[ja4BrowserCorpusSnapshot]

func init() {
	if snap := parseJA4BrowserCorpus(ja4BrowserCorpusEmbed); snap != nil {
		ja4BrowserCorpusActive.Store(snap)
	}
}

func PublishJA4BrowserCorpus(snap *ja4BrowserCorpusSnapshot) {
	if snap == nil || len(snap.prefixFamilies) == 0 {
		return
	}
	ja4BrowserCorpusActive.Store(snap)
}

func loadJA4BrowserCorpusFromDir(dir string) *ja4BrowserCorpusSnapshot {
	base := parseJA4BrowserCorpus(ja4BrowserCorpusEmbed)
	if dir == "" {
		return base
	}
	data, err := os.ReadFile(filepath.Join(dir, "ja4_browser_corpus.txt"))
	if err != nil || len(data) == 0 {
		return base
	}
	overlay := parseJA4BrowserCorpus(data)
	if overlay == nil {
		return base
	}
	if base == nil || len(base.prefixFamilies) == 0 {
		return overlay
	}
	merged := &ja4BrowserCorpusSnapshot{
		prefixFamilies: make(map[string]uint8, len(base.prefixFamilies)+len(overlay.prefixFamilies)),
	}
	for k, v := range base.prefixFamilies {
		merged.prefixFamilies[k] = v
	}
	for k, v := range overlay.prefixFamilies {
		merged.prefixFamilies[k] = v
	}
	return merged
}

func parseJA4BrowserCorpus(data []byte) *ja4BrowserCorpusSnapshot {
	if len(data) == 0 {
		return nil
	}
	entries := make(map[string]uint8)
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
		prefix := strings.ToLower(strings.TrimSpace(line[:eq]))
		if len(prefix) < 4 || len(prefix) > 32 {
			continue
		}
		mask := parseTLSBrowserFamilyMask(strings.TrimSpace(line[eq+1:]))
		if mask == 0 {
			continue
		}
		entries[prefix] = mask
	}
	if len(entries) == 0 {
		return nil
	}
	return &ja4BrowserCorpusSnapshot{prefixFamilies: entries}
}

func parseTLSBrowserFamilyMask(raw string) uint8 {
	var mask uint8
	start := 0
	n := len(raw)
	for i := 0; i <= n; i++ {
		if i < n && raw[i] != ',' {
			continue
		}
		token := strings.ToLower(strings.TrimSpace(raw[start:i]))
		switch token {
		case "chrome":
			mask |= tlsBrowserChrome
		case "firefox":
			mask |= tlsBrowserFirefox
		case "safari":
			mask |= tlsBrowserSafari
		case "okhttp":
			mask |= tlsBrowserOkhttp
		case "go":
			mask |= tlsBrowserGo
		}
		start = i + 1
	}
	return mask
}

func ja4PrefixBytes(ja4 []byte) []byte {
	if len(ja4) == 0 {
		return nil
	}
	for i := 0; i < len(ja4); i++ {
		if ja4[i] == '_' {
			return ja4[:i]
		}
	}
	return ja4
}

func classifyTLSBrowserFamily(ua string) uint8 {
	if ua == "" || uaMatchesInAppWebView(ua) {
		return 0
	}
	n := len(ua)
	if n > uaScanMax {
		n = uaScanMax
	}
	for i := 0; i < n; i++ {
		if i+6 <= n && matchUAAt(ua, i, n, "OkHttp") {
			return tlsBrowserOkhttp
		}
	}
	if uaClaimsChromeNotChromium(ua) {
		return tlsBrowserChrome
	}
	for i := 0; i < n; i++ {
		if i+7 <= n && matchUAAt(ua, i, n, "Firefox") {
			return tlsBrowserFirefox
		}
	}
	for i := 0; i < n; i++ {
		if i+6 <= n && matchUAAt(ua, i, n, "Safari") {
			if !uaContainsChromeToken(ua, n) {
				return tlsBrowserSafari
			}
		}
	}
	return 0
}

func uaContainsChromeToken(ua string, n int) bool {
	for i := 0; i < n; i++ {
		if i+6 <= n && matchUAAt(ua, i, n, "Chrome") {
			return true
		}
	}
	return false
}

func ja4BrowserCorpusMismatch(ua string, ja4 []byte) bool {
	if len(ja4) == 0 || len(ja4) > tlsFingerprintMaxLen {
		return false
	}
	snap := ja4BrowserCorpusActive.Load()
	if snap == nil || len(snap.prefixFamilies) == 0 {
		return false
	}
	prefix := ja4PrefixBytes(ja4)
	if len(prefix) == 0 {
		return false
	}
	allowed, ok := snap.prefixFamilies[string(prefix)]
	if !ok {
		return false
	}
	family := classifyTLSBrowserFamily(ua)
	if family == 0 {
		return false
	}
	return allowed&family == 0
}

package netintel

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
)

type dcASNFeedLoader struct {
	dir     string
	refresh time.Duration
	table   *DCASNTable
	gen     atomic.Uint64
}

func NewDCASNFeedLoader(cfg *config.Config, table *DCASNTable) *dcASNFeedLoader {
	if cfg == nil || !cfg.DCASNHotEnabled || table == nil {
		return nil
	}
	dir := cfg.DCASNFeedDir
	if dir == "" {
		dir = "/var/lib/ad-event-processor/dc-asn"
	}
	return &dcASNFeedLoader{
		dir:     dir,
		refresh: cfg.DCASNFeedRefresh,
		table:   table,
	}
}

func (l *dcASNFeedLoader) Start(ctx context.Context) {
	l.refreshOnce()
	ticker := time.NewTicker(l.refresh)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.refreshOnce()
		}
	}
}

func (l *dcASNFeedLoader) refreshOnce() {
	path := filepath.Join(l.dir, "dc_asn.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		metrics.DCASNFeedRefreshErrorsTotal.Inc()
		if l.table.Ready() {
			return
		}
		metrics.DCASNHotUninitialized.Set(1)
		return
	}
	asns := parseDCASNFeed(data)
	gen := l.gen.Add(1)
	l.table.Publish(buildDCASNSnapshot(asns, gen))
	metrics.DCASNFeedRefreshTotal.Inc()
	metrics.DCASNHotEntries.Set(float64(len(asns)))
	metrics.DCASNHotUninitialized.Set(0)
}

var mobileASNDenylist = map[uint32]struct{}{
	3215:  {},
	12322: {},
}

type dcASNSnapshot struct {
	gen uint64
	asn map[uint32]struct{}
}

type DCASNTable struct {
	active atomic.Pointer[dcASNSnapshot]
}

func NewDCASNTable() *DCASNTable {
	return &DCASNTable{}
}

func (t *DCASNTable) Publish(snap *dcASNSnapshot) {
	if t == nil || snap == nil {
		return
	}
	t.active.Store(snap)
}

func (t *DCASNTable) Ready() bool {
	return t != nil && t.active.Load() != nil
}

func (t *DCASNTable) Size() int {
	snap := t.active.Load()
	if snap == nil {
		return 0
	}
	return len(snap.asn)
}

func (t *DCASNTable) IsDatacenter(asn uint32) bool {
	if asn == 0 {
		return false
	}
	if _, denied := mobileASNDenylist[asn]; denied {
		return false
	}
	snap := t.active.Load()
	if snap == nil {
		return false
	}
	_, ok := snap.asn[asn]
	return ok
}

func BuildDCASNSnapshot(asns map[uint32]struct{}, gen uint64) *DCASNSnapshot {
	return buildDCASNSnapshot(asns, gen)
}

type DCASNSnapshot = dcASNSnapshot

func buildDCASNSnapshot(asns map[uint32]struct{}, gen uint64) *dcASNSnapshot {
	if len(asns) == 0 {
		return &dcASNSnapshot{gen: gen, asn: map[uint32]struct{}{}}
	}
	dup := make(map[uint32]struct{}, len(asns))
	for asn := range asns {
		dup[asn] = struct{}{}
	}
	return &dcASNSnapshot{gen: gen, asn: dup}
}

func parseDCASNFeed(data []byte) map[uint32]struct{} {
	out := make(map[uint32]struct{})
	for line := range splitLineIter(data) {
		asn, ok := parseASNLine(line)
		if ok {
			out[asn] = struct{}{}
		}
	}
	return out
}

func parseASNLine(line string) (uint32, bool) {
	return ParseASNLine(line)
}

func ParseASNLine(line string) (uint32, bool) {
	if line == "" {
		return 0, false
	}
	if len(line) > 2 && (line[0] == 'A' || line[0] == 'a') && (line[1] == 'S' || line[1] == 's') {
		line = line[2:]
	}
	var val uint32
	for i := range len(line) {
		c := line[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		if val > 655350 {
			return 0, false
		}
		val = val*10 + uint32(c-'0')
	}
	if val == 0 {
		return 0, false
	}
	return val, true
}

func splitLineIter(data []byte) func(func(string) bool) {
	s := string(data)
	return func(yield func(string) bool) {
		start := 0
		for i := 0; i <= len(s); i++ {
			if i == len(s) || s[i] == '\n' {
				line := trimASCII(s[start:i])
				if line != "" && !stringsHasPrefixASCII(line, "#") {
					if !yield(line) {
						return
					}
				}
				start = i + 1
			}
		}
	}
}

func stringsHasPrefixASCII(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

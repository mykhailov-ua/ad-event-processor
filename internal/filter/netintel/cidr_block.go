package netintel

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/bits"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/coldpath"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	cidrFormatLines = iota
	cidrFormatAWSGCP
	cidrFormatAzure
)

type cidrFeedSource struct {
	feed   uint8
	name   string
	file   string
	url    string
	format int
}

type cidrFeedLoader struct {
	dir          string
	refresh      time.Duration
	download     bool
	sources      []cidrFeedSource
	table        *CIDRTable
	httpClient   *http.Client
	errCounters  [CIDRFeedCount]prometheus.Counter
	gen          atomic.Uint64
	lastPrefixes atomic.Int64
}

func NewCIDRFeedLoader(cfg *config.Config, table *CIDRTable) *cidrFeedLoader {
	if cfg == nil || !cfg.CIDRBlockEnabled || table == nil {
		return nil
	}
	l := &cidrFeedLoader{
		dir:        cfg.CIDRFeedDir,
		refresh:    cfg.CIDRFeedRefresh,
		download:   cfg.CIDRFeedDownloadEnable,
		table:      table,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		sources: []cidrFeedSource{
			{feed: CIDRFeedAWS, name: "aws", file: "aws.json", url: cfg.CIDRFeedURLAWS, format: cidrFormatAWSGCP},
			{feed: CIDRFeedGCP, name: "gcp", file: "gcp.json", url: cfg.CIDRFeedURLGCP, format: cidrFormatAWSGCP},
			{feed: CIDRFeedAzure, name: "azure", file: "azure.json", url: cfg.CIDRFeedURLAzure, format: cidrFormatAzure},
			{feed: CIDRFeedTor, name: "tor", file: "tor.txt", url: cfg.CIDRFeedURLTor, format: cidrFormatLines},
			{feed: CIDRFeedOther, name: "other", file: "other.txt", format: cidrFormatLines},
		},
	}
	if l.refresh <= 0 {
		l.refresh = 24 * time.Hour
	}
	for i := range l.errCounters {
		l.errCounters[i] = metrics.CIDRFeedRefreshErrorsTotal.WithLabelValues(CIDRFeedNames[i])
	}
	return l
}

func (l *cidrFeedLoader) Start(ctx context.Context) {
	l.refreshOnce(ctx)
	ticker := time.NewTicker(l.refresh)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.refreshOnce(ctx)
		}
	}
}

func (l *cidrFeedLoader) refreshOnce(ctx context.Context) {
	var b cidrBuilder
	root4, root6 := int32(cidrNoIndex), int32(cidrNoIndex)
	okFeeds := 0

	for _, src := range l.sources {
		if l.download && src.url != "" {
			if err := l.fetch(ctx, src); err != nil {
				l.errCounters[src.feed].Inc()
				slog.Warn("cidr feed download failed, using cache", "feed", src.name, "error", err)
			}
		}
		n, err := l.parseFeed(src, &b, &root4, &root6)
		if err != nil {
			l.errCounters[src.feed].Inc()
			slog.Warn("cidr feed parse failed", "feed", src.name, "error", err)
			continue
		}
		if n > 0 {
			okFeeds++
		}
	}

	if len(b.prefs) == 0 {
		if !l.table.Ready() {
			metrics.CIDRLPMUninitialized.Set(1)
			slog.Warn("cidr l1 table uninitialized (no feed data); L1 fail-open", "dir", l.dir)
		}
		return
	}

	gen := l.gen.Add(1)
	l.table.Publish(b.snapshot(root4, root6, gen))
	l.lastPrefixes.Store(int64(len(b.prefs)))
	metrics.CIDRLPMUninitialized.Set(0)
	metrics.CIDRLPMPrefixes.Set(float64(len(b.prefs)))
	metrics.CIDRFeedRefreshTotal.Inc()
	slog.Info("cidr l1 snapshot published", "prefixes", len(b.prefs), "nodes", len(b.nodes), "feeds_ok", okFeeds, "gen", gen)
}

func (l *cidrFeedLoader) fetch(ctx context.Context, src cidrFeedSource) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := l.httpClient.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	if err := os.MkdirAll(l.dir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(l.dir, src.file+".tmp")
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, io.LimitReader(resp.Body, 64<<20))
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, filepath.Join(l.dir, src.file))
}

func (l *cidrFeedLoader) parseFeed(src cidrFeedSource, b *cidrBuilder, root4, root6 *int32) (int, error) {
	f, err := os.Open(filepath.Join(l.dir, src.file))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer func() { _ = f.Close() }()
	switch src.format {
	case cidrFormatLines:
		return parseCIDRLines(f, src.feed, b, root4, root6)
	case cidrFormatAzure:
		return parseCIDRAzure(f, src.feed, b, root4, root6)
	default:
		return parseCIDRAWSGCP(f, src.feed, b, root4, root6)
	}
}

func parseCIDRLines(r io.Reader, feed uint8, b *cidrBuilder, root4, root6 *int32) (int, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	n := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		p, err := cidrParseEntry(line)
		if err != nil {
			continue
		}
		b.addPrefix(p, feed, root4, root6)
		n++
	}
	return n, sc.Err()
}

func cidrParseEntry(s string) (netip.Prefix, error) {
	if p, err := netip.ParsePrefix(s); err == nil {
		return p.Masked(), nil
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Prefix{}, err
	}
	if a.Zone() != "" {
		return netip.Prefix{}, fmt.Errorf("zoned address %q not allowed in feeds", s)
	}
	bits := 32
	if a.Is6() {
		bits = 128
	}
	return netip.PrefixFrom(a, bits), nil
}

type cidrPrefixListJSON struct {
	Prefixes []struct {
		IPPrefix   string `json:"ip_prefix"`
		IPv6Prefix string `json:"ipv6_prefix"`
	} `json:"prefixes"`
}

func parseCIDRAWSGCP(r io.Reader, feed uint8, b *cidrBuilder, root4, root6 *int32) (int, error) {
	var doc cidrPrefixListJSON
	if err := json.NewDecoder(io.LimitReader(r, 64<<20)).Decode(&doc); err != nil {
		return 0, err
	}
	n := 0
	for _, e := range doc.Prefixes {
		raw := e.IPPrefix
		if raw == "" {
			raw = e.IPv6Prefix
		}
		if raw == "" {
			continue
		}
		p, err := netip.ParsePrefix(raw)
		if err != nil {
			continue
		}
		b.addPrefix(p.Masked(), feed, root4, root6)
		n++
	}
	return n, nil
}

type cidrAzureJSON struct {
	Values []struct {
		Properties struct {
			AddressPrefixes []string `json:"addressPrefixes"`
		} `json:"properties"`
	} `json:"values"`
}

func parseCIDRAzure(r io.Reader, feed uint8, b *cidrBuilder, root4, root6 *int32) (int, error) {
	var doc cidrAzureJSON
	if err := json.NewDecoder(io.LimitReader(r, 64<<20)).Decode(&doc); err != nil {
		return 0, err
	}
	n := 0
	for _, v := range doc.Values {
		for _, raw := range v.Properties.AddressPrefixes {
			p, err := netip.ParsePrefix(raw)
			if err != nil {
				continue
			}
			b.addPrefix(p.Masked(), feed, root4, root6)
			n++
		}
	}
	return n, nil
}

const (
	CIDRFeedAWS uint8 = iota
	CIDRFeedGCP
	CIDRFeedAzure
	CIDRFeedTor
	CIDRFeedOther
	CIDRFeedCount
)

var CIDRFeedNames = [CIDRFeedCount]string{"aws", "gcp", "azure", "tor", "other"}

const cidrNoIndex = -1

type CIDRNode struct {
	child   [2]int32
	pref    int32
	critbit int32
}

type cidrPrefix struct {
	addr [16]byte
	bits uint8
	feed uint8
}

type cidrSnapshot struct {
	gen   uint64
	root4 int32
	root6 int32
	nodes []CIDRNode
	prefs []cidrPrefix
}

// CIDRTable: immutable radix snapshot (atomic.Pointer); MatchIP is zero-alloc on hot filter path.
type CIDRTable struct {
	active atomic.Pointer[cidrSnapshot]
}

func NewCIDRTable() *CIDRTable {
	return &CIDRTable{}
}

func (t *CIDRTable) Publish(s *cidrSnapshot) {
	t.active.Store(s)
}

func (t *CIDRTable) Ready() bool {
	return t.active.Load() != nil
}

func (t *CIDRTable) SnapshotSize() (prefixes int, gen uint64, ok bool) {
	snap := t.active.Load()
	if snap == nil {
		return 0, 0, false
	}
	return len(snap.prefs), snap.gen, true
}

func (t *CIDRTable) Match4(ip [4]byte) (bool, uint8) {
	snap := t.active.Load()
	if snap == nil || snap.root4 < 0 {
		return false, 0
	}
	var key [16]byte
	copy(key[:4], ip[:])
	return snap.lookup(snap.root4, &key)
}

func (t *CIDRTable) Match6(ip [16]byte) (bool, uint8) {
	snap := t.active.Load()
	if snap == nil || snap.root6 < 0 {
		return false, 0
	}
	return snap.lookup(snap.root6, &ip)
}

func (t *CIDRTable) MatchIP(ipStr string) (bool, uint8) {
	snap := t.active.Load()
	if snap == nil {
		return false, 0
	}
	a, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false, 0
	}
	if a.Is4() || a.Is4In6() {
		a4 := a.As4()
		if snap.root4 < 0 {
			return false, 0
		}
		var key [16]byte
		copy(key[:4], a4[:])
		return snap.lookup(snap.root4, &key)
	}
	if snap.root6 < 0 {
		return false, 0
	}
	key := a.As16()
	return snap.lookup(snap.root6, &key)
}

func (s *cidrSnapshot) lookup(root int32, key *[16]byte) (bool, uint8) {
	nodes := s.nodes
	prefs := s.prefs
	_ = nodes[len(nodes)-1]
	cur := root
	for cur >= 0 {
		n := &nodes[cur]
		if n.pref >= 0 {
			p := &prefs[n.pref]
			if cidrPrefixMatch(key, &p.addr, p.bits) {
				return true, p.feed
			}
		}
		if n.critbit < 0 {
			return false, 0
		}
		cur = n.child[cidrBitAt(key, n.critbit)]
	}
	return false, 0
}

func cidrBitAt(key *[16]byte, i int32) uint32 {
	return uint32(key[i>>3]>>(7-(i&7))) & 1
}

func cidrPrefixMatch(key, addr *[16]byte, nbits uint8) bool {
	n := int(nbits)
	full := n >> 3
	for i := range full {
		if key[i] != addr[i] {
			return false
		}
	}
	rem := n & 7
	if rem == 0 {
		return true
	}
	mask := byte(0xff << (8 - rem))
	return key[full]&mask == addr[full]&mask
}

func cidrFirstDiff(a *[16]byte, abits uint8, b *[16]byte, bbits uint8) int32 {
	n := abits
	if bbits < n {
		n = bbits
	}
	full := int(n) >> 3
	for i := range full {
		if a[i] != b[i] {
			return int32(i*8 + bits.LeadingZeros8(a[i]^b[i]))
		}
	}
	rem := n & 7
	if rem > 0 {
		mask := byte(0xff << (8 - rem))
		if x := (a[full] ^ b[full]) & mask; x != 0 {
			return int32(full*8 + bits.LeadingZeros8(x))
		}
	}
	return -1
}

type cidrBuilder struct {
	nodes []CIDRNode
	prefs []cidrPrefix
}

func (b *cidrBuilder) addPrefix(p netip.Prefix, feed uint8, root4, root6 *int32) {
	if p.Addr().Is4() {
		a4 := p.Addr().As4()
		var key [16]byte
		copy(key[:4], a4[:])
		b.insert(root4, key, uint8(p.Bits()), feed)
		return
	}
	b.insert(root6, p.Addr().As16(), uint8(p.Bits()), feed)
}

func (b *cidrBuilder) addLeaf(pref int32) int32 {
	b.nodes = append(b.nodes, CIDRNode{
		child:   [2]int32{cidrNoIndex, cidrNoIndex},
		pref:    pref,
		critbit: -1,
	})
	return int32(len(b.nodes) - 1)
}

func (b *cidrBuilder) repPrefix(cur int32) cidrPrefix {
	for b.nodes[cur].critbit >= 0 {
		next := b.nodes[cur].child[0]
		if next < 0 {
			next = b.nodes[cur].child[1]
		}
		cur = next
	}
	return b.prefs[b.nodes[cur].pref]
}

func (b *cidrBuilder) insert(root *int32, addr [16]byte, nbits uint8, feed uint8) {
	b.Insert(root, addr, nbits, feed)
}

func (b *CIDRBuilder) Insert(root *int32, addr [16]byte, nbits uint8, feed uint8) {
	b.insertImpl(root, addr, nbits, feed)
}

func (b *cidrBuilder) insertImpl(root *int32, addr [16]byte, nbits uint8, feed uint8) {
	prefIdx := int32(len(b.prefs))
	b.prefs = append(b.prefs, cidrPrefix{addr: addr, bits: nbits, feed: feed})

	if *root < 0 {
		*root = b.addLeaf(prefIdx)
		return
	}

	parent := int32(cidrNoIndex)
	var pdir uint32
	cur := *root
	missing := false
	for b.nodes[cur].critbit >= 0 && b.nodes[cur].critbit < int32(nbits) {
		parent = cur
		pdir = cidrBitAt(&addr, b.nodes[cur].critbit)
		next := b.nodes[cur].child[pdir]
		if next < 0 {
			missing = true
			break
		}
		cur = next
	}

	rep := b.repPrefix(cur)
	d := cidrFirstDiff(&addr, nbits, &rep.addr, rep.bits)
	if missing {
		if d < 0 || d >= b.nodes[cur].critbit {
			b.nodes[cur].child[pdir] = b.addLeaf(prefIdx)
			return
		}
	}
	if d < 0 {
		switch {
		case nbits == rep.bits:
			b.prefs = b.prefs[:prefIdx]
			return
		case nbits < rep.bits:
			if b.nodes[cur].critbit == int32(nbits) {
				if b.nodes[cur].pref < 0 {
					b.nodes[cur].pref = prefIdx
				} else {
					b.prefs = b.prefs[:prefIdx]
				}
				return
			}
			d = int32(nbits)
		default:

			repLeaf := b.nodes[cur]
			split := b.addNode(int32(rep.bits), repLeaf.pref)
			b.nodes[split].child[cidrBitAt(&addr, int32(rep.bits))] = b.addLeaf(prefIdx)
			b.linkChild(root, parent, pdir, split)
			return
		}
	}

	if d >= int32(nbits) {

		split := b.addNode(d, prefIdx)
		b.nodes[split].child[cidrBitAt(&rep.addr, d)] = cur
		b.linkChild(root, parent, pdir, split)
		return
	}

	walk := *root
	wparent := int32(cidrNoIndex)
	var wpdir uint32
	for b.nodes[walk].critbit >= 0 && b.nodes[walk].critbit < d {
		wparent = walk
		wpdir = cidrBitAt(&addr, b.nodes[walk].critbit)
		walk = b.nodes[walk].child[wpdir]
	}
	split := b.addNode(d, cidrNoIndex)
	kdir := cidrBitAt(&addr, d)
	b.nodes[split].child[kdir] = b.addLeaf(prefIdx)
	b.nodes[split].child[1-kdir] = walk
	if wparent < 0 {
		*root = split
		return
	}
	b.nodes[wparent].child[wpdir] = split
}

func (b *cidrBuilder) snapshot(root4, root6 int32, gen uint64) *cidrSnapshot {
	return b.Snapshot(root4, root6, gen)
}

func (b *CIDRBuilder) Snapshot(root4, root6 int32, gen uint64) *CIDRSnapshot {
	return &cidrSnapshot{
		gen:   gen,
		root4: root4,
		root6: root6,
		nodes: b.nodes,
		prefs: b.prefs,
	}
}

type CIDRSnapshot = cidrSnapshot

func (b *cidrBuilder) addNode(critbit, pref int32) int32 {
	b.nodes = append(b.nodes, CIDRNode{
		child:   [2]int32{cidrNoIndex, cidrNoIndex},
		pref:    pref,
		critbit: critbit,
	})
	return int32(len(b.nodes) - 1)
}

func (b *cidrBuilder) linkChild(root *int32, parent int32, pdir uint32, node int32) {
	if parent < 0 {
		*root = node
		return
	}
	b.nodes[parent].child[pdir] = node
}

type CIDRBuilder = cidrBuilder

const CIDRNoIndex = cidrNoIndex

func BuildCIDRTableFromPrefixes(cidrs ...string) (*CIDRTable, error) {
	var b cidrBuilder
	root4, root6 := int32(cidrNoIndex), int32(cidrNoIndex)
	for _, s := range cidrs {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return nil, err
		}
		p = p.Masked()
		if p.Addr().Is4() {
			a4 := p.Addr().As4()
			var key [16]byte
			copy(key[:4], a4[:])
			b.insert(&root4, key, uint8(p.Bits()), CIDRFeedAWS)
		} else {
			b.insert(&root6, p.Addr().As16(), uint8(p.Bits()), CIDRFeedTor)
		}
	}
	table := NewCIDRTable()
	table.Publish(b.snapshot(root4, root6, 1))
	return table, nil
}

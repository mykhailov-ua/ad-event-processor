package ingestion

import (
	"bufio"
	"context"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/metrics"
)

type proxyVPNFeedLoader struct {
	dir     string
	refresh time.Duration
	table   *ProxyVPNTable
	gen     atomic.Uint64
}

func NewProxyVPNFeedLoader(cfg *config.Config, table *ProxyVPNTable) *proxyVPNFeedLoader {
	if cfg == nil || !cfg.ProxyVPNL15Enabled || table == nil {
		return nil
	}
	l := &proxyVPNFeedLoader{
		dir:     cfg.ProxyVPNFeedDir,
		refresh: cfg.ProxyVPNFeedRefresh,
		table:   table,
	}
	if l.refresh <= 0 {
		l.refresh = 24 * time.Hour
	}
	if l.dir == "" {
		l.dir = "/var/lib/ad-event-processor/proxy-vpn"
	}
	return l
}

func (l *proxyVPNFeedLoader) Start(ctx context.Context) {
	if l == nil {
		return
	}
	l.reloadOnce()
	ticker := time.NewTicker(l.refresh)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.reloadOnce()
		}
	}
}

func (l *proxyVPNFeedLoader) reloadOnce() {
	path := filepath.Join(l.dir, "proxy_vpn.txt")
	f, err := os.Open(path)
	if err != nil {
		metrics.ProxyVPNFeedRefreshErrorsTotal.Inc()
		slog.Warn("proxy vpn feed open failed", "path", path, "error", err)
		return
	}
	defer func() { _ = f.Close() }()

	var b proxyVPNBuilder
	root4, root6 := int32(cidrNoIndex), int32(cidrNoIndex)
	sc := bufio.NewScanner(f)
	lines := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		prefix, connType, asn, ok := parseProxyVPNFeedLine(line)
		if !ok {
			continue
		}
		b.addPrefix(prefix, connType, asn, &root4, &root6)
		lines++
	}
	if err := sc.Err(); err != nil {
		metrics.ProxyVPNFeedRefreshErrorsTotal.Inc()
		slog.Warn("proxy vpn feed scan failed", "path", path, "error", err)
		return
	}
	if lines == 0 {
		metrics.ProxyVPNLPMUninitialized.Set(1)
		return
	}
	gen := l.gen.Add(1)
	snap := b.snapshot(root4, root6, gen)
	l.table.Publish(snap)
	metrics.ProxyVPNFeedRefreshTotal.Inc()
	metrics.ProxyVPNLPMPrefixes.Set(float64(len(snap.prefs)))
	metrics.ProxyVPNLPMUninitialized.Set(0)
	slog.Info("proxy vpn feed published", "prefixes", len(snap.prefs), "gen", gen)
}

func parseProxyVPNFeedLine(line string) (netip.Prefix, uint8, uint32, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return netip.Prefix{}, 0, 0, false
	}
	prefix, err := netip.ParsePrefix(fields[0])
	if err != nil {
		return netip.Prefix{}, 0, 0, false
	}
	if !prefix.IsValid() {
		return netip.Prefix{}, 0, 0, false
	}
	prefix = prefix.Masked()
	var asn uint32
	if fields[1] != "-" && fields[1] != "0" {
		n, err := strconv.ParseUint(fields[1], 10, 32)
		if err != nil {
			return netip.Prefix{}, 0, 0, false
		}
		asn = uint32(n)
	}
	flags := ""
	if len(fields) > 2 {
		flags = strings.Join(fields[2:], " ")
	}
	connType := parseProxyVPNConnFlags(flags)
	if connType == 0 {
		connType = ProxyVPNConnVPN | ProxyVPNConnHosting
	}
	return prefix, connType, asn, true
}

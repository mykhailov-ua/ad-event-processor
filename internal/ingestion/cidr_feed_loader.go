package ingestion

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
	if cfg == nil || !cfg.CIDRL1Enabled || table == nil {
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
		l.errCounters[i] = metrics.CIDRFeedRefreshErrorsTotal.WithLabelValues(cidrFeedNames[i])
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

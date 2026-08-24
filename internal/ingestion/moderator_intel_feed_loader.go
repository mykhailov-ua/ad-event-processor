package ingestion

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/moderatorintel"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	moderatorIntelFeedFile = "moderator_intel_v1.json"
	moderatorIntelSigFile  = "moderator_intel_v1.sig"
)

type moderatorIntelFeedLoader struct {
	dir           string
	refresh       time.Duration
	download      bool
	feedURL       string
	secret        []byte
	allowUnsigned bool
	table         *ModeratorIPTable
	httpClient    *http.Client
	gen           atomic.Uint64
	errCounter    prometheus.Counter
}

func NewModeratorIntelFeedLoader(cfg *config.Config, table *ModeratorIPTable) *moderatorIntelFeedLoader {
	if cfg == nil || !cfg.ModeratorIntelEnabled || table == nil {
		return nil
	}
	l := &moderatorIntelFeedLoader{
		dir:           cfg.ModeratorIntelFeedDir,
		refresh:       cfg.ModeratorIntelFeedRefresh,
		download:      cfg.ModeratorIntelFeedDownload,
		feedURL:       strings.TrimSpace(cfg.ModeratorIntelFeedURL),
		secret:        []byte(strings.TrimSpace(cfg.ModeratorIntelFeedSecret)),
		allowUnsigned: cfg.ModeratorIntelAllowUnsigned,
		table:         table,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		errCounter:    metrics.ModeratorIntelFeedRefreshErrorsTotal,
	}
	if l.refresh <= 0 {
		l.refresh = 24 * time.Hour
	}
	if l.dir == "" {
		l.dir = "/var/lib/ad-event-processor/moderator-intel"
	}
	return l
}

func (l *moderatorIntelFeedLoader) Start(ctx context.Context) {
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

func (l *moderatorIntelFeedLoader) refreshOnce(ctx context.Context) {
	if l.download && l.feedURL != "" {
		if err := l.fetch(ctx); err != nil {
			l.errCounter.Inc()
			slog.Warn("moderator intel feed download failed, using cache", "error", err)
		}
	}
	body, err := os.ReadFile(filepath.Join(l.dir, moderatorIntelFeedFile))
	if err != nil {
		if !l.table.Ready() {
			metrics.ModeratorIntelLPMUninitialized.Set(1)
		}
		if !os.IsNotExist(err) {
			l.errCounter.Inc()
			slog.Warn("moderator intel feed open failed", "error", err)
		}
		return
	}
	if len(l.secret) > 0 {
		sigBytes, err := os.ReadFile(filepath.Join(l.dir, moderatorIntelSigFile))
		if err != nil {
			l.errCounter.Inc()
			slog.Warn("moderator intel feed signature missing", "error", err)
			return
		}
		if !moderatorintel.VerifySignature(l.secret, body, strings.TrimSpace(string(sigBytes))) {
			l.errCounter.Inc()
			slog.Warn("moderator intel feed signature invalid")
			return
		}
	} else if !l.allowUnsigned {
		l.errCounter.Inc()
		slog.Warn("moderator intel feed secret not configured; refusing unsigned feed")
		return
	}
	feed, err := moderatorintel.ParseFeedV1(body, time.Now().UTC())
	if err != nil {
		l.errCounter.Inc()
		slog.Warn("moderator intel feed parse failed; retaining previous snapshot", "error", err)
		return
	}
	if len(feed.Entries) == 0 {
		l.errCounter.Inc()
		return
	}
	gen := l.gen.Add(1)
	l.table.publishEntries(feed.Entries, gen)
	metrics.ModeratorIntelFeedRefreshTotal.Inc()
	metrics.ModeratorIntelLPMUninitialized.Set(0)
	metrics.ModeratorIntelLPMPrefixes.Set(float64(len(feed.Entries)))
	slog.Info("moderator intel snapshot published",
		"source", feed.Source,
		"prefixes", len(feed.Entries),
		"expires_at", feed.ExpiresAt.UTC().Format(time.RFC3339),
		"gen", gen)
}

func (l *moderatorIntelFeedLoader) fetch(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.feedURL, http.NoBody)
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
		return errHTTPStatus(resp.StatusCode)
	}
	if err := os.MkdirAll(l.dir, 0o755); err != nil {
		return err
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(l.dir, moderatorIntelFeedFile), body, 0o644); err != nil {
		return err
	}
	if sig := strings.TrimSpace(resp.Header.Get("X-Feed-Signature")); sig != "" {
		return os.WriteFile(filepath.Join(l.dir, moderatorIntelSigFile), []byte(sig), 0o644)
	}
	return nil
}

type errHTTPStatus int

func (e errHTTPStatus) Error() string {
	return "http " + strconv.Itoa(int(e))
}

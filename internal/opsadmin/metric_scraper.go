package opsadmin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

var opsScrapeMetricNames = map[string]struct{}{
	"ad_http_requests_total":          {},
	"ad_recon_drift_micro":            {},
	"ad_control_outbox_pending_total": {},
	"ad_tracker_redis_shard_healthy":  {},
}

type scrapedMetric struct {
	Name       string
	LabelsHash string
	Value      float64
}

type MetricScraper struct {
	host      OpsMetricScraperHost
	pool      *pgxpool.Pool
	url       string
	client    *http.Client
	interval  time.Duration
	retention time.Duration
	fetch     func(ctx context.Context, url string) ([]byte, string, error)
}

func NewMetricScraper(host OpsMetricScraperHost, scrapeURL string) *MetricScraper {
	if scrapeURL == "" {
		scrapeURL = "http://127.0.0.1:8188/metrics"
	}
	client := &http.Client{Timeout: defaultOpsMetricScrapeTimeout}
	w := &MetricScraper{
		host:      host,
		interval:  defaultOpsMetricScrapeInterval,
		retention: defaultOpsMetricRetention,
		client:    client,
		url:       scrapeURL,
	}
	if host != nil {
		w.pool = host.GetPool()
	}
	w.fetch = func(ctx context.Context, url string) ([]byte, string, error) {
		return fetchMetrics(ctx, client, url)
	}
	return w
}

func StartMetricScraper(host OpsMetricScraperHost, ctx context.Context, scrapeURL string) {
	if host == nil || host.GetPool() == nil {
		return
	}
	w := NewMetricScraper(host, scrapeURL)
	host.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
}

func fetchMetrics(ctx context.Context, client *http.Client, url string) ([]byte, string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("metrics scrape status %d", resp.StatusCode)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func (w *MetricScraper) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	slog.Info("ops metric scraper starting", "url", w.url, "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.tick(ctx, time.Now().UTC())
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			w.tick(ctx, t.UTC())
		}
	}
}

func (w *MetricScraper) tick(ctx context.Context, now time.Time) {
	run := func(runCtx context.Context) error {
		if err := w.scrapeAndStore(runCtx, now); err != nil {
			return err
		}
		return w.expireSamples(runCtx, now)
	}
	if w.host != nil {
		if err := w.host.WithPostgresLow(ctx, run); err != nil {
			slog.Error("ops metric scraper tick failed", "error", err)
		}
		return
	}
	if err := run(ctx); err != nil {
		slog.Error("ops metric scraper tick failed", "error", err)
	}
}

func (w *MetricScraper) scrapeAndStore(ctx context.Context, now time.Time) error {
	fetch := w.fetch
	if fetch == nil {
		fetch = func(ctx context.Context, url string) ([]byte, string, error) {
			return fetchMetrics(ctx, w.client, url)
		}
	}
	body, contentType, err := fetch(ctx, w.url)
	if err != nil {
		return fmt.Errorf("fetch metrics: %w", err)
	}
	samples, err := parsePrometheusMetrics(bytes.NewReader(body), contentType)
	if err != nil {
		return fmt.Errorf("parse metrics: %w", err)
	}
	if len(samples) == 0 {
		return nil
	}

	const insertSQL = insertOpsMetricSampleSQL
	ts := pgtype.Timestamptz{Time: now, Valid: true}
	batch := &pgx.Batch{}
	for _, sample := range samples {
		batch.Queue(insertSQL, sample.Name, sample.LabelsHash, ts, sample.Value)
	}
	br := w.pool.SendBatch(ctx, batch)
	var batchErr error
	for range samples {
		if _, err := br.Exec(); err != nil && batchErr == nil {
			batchErr = fmt.Errorf("insert metric sample batch: %w", err)
		}
	}
	if closeErr := br.Close(); closeErr != nil && batchErr == nil {
		batchErr = fmt.Errorf("close metric batch: %w", closeErr)
	}
	return batchErr
}

func (w *MetricScraper) expireSamples(ctx context.Context, now time.Time) error {
	cutoff := now.Add(-w.retention)
	q := db.New(w.pool)
	if _, err := q.DeleteExpiredOpsMetricSamples(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true}); err != nil {
		return fmt.Errorf("expire metric samples: %w", err)
	}
	return nil
}

func parsePrometheusMetrics(r io.Reader, contentType string) ([]scrapedMetric, error) {
	format := expfmt.NewFormat(expfmt.TypeTextPlain)
	if contentType != "" {
		if parsed := expfmt.ResponseFormat(http.Header{"Content-Type": {contentType}}); parsed != expfmt.FmtUnknown {
			format = parsed
		}
	}
	dec := expfmt.NewDecoder(r, format)
	var out []scrapedMetric
	var maxDrift float64
	var sawDrift bool
	for {
		var mf dto.MetricFamily
		if err := dec.Decode(&mf); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if mf.Name == nil {
			continue
		}
		name := *mf.Name
		if _, ok := opsScrapeMetricNames[name]; !ok {
			continue
		}
		for _, m := range mf.Metric {
			val, ok := metricValue(m)
			if !ok {
				continue
			}
			if name == "ad_recon_drift_micro" {
				sawDrift = true
				if val > maxDrift {
					maxDrift = val
				}
				continue
			}
			out = append(out, scrapedMetric{
				Name:       name,
				LabelsHash: labelsHash(m.Label),
				Value:      val,
			})
		}
	}
	if sawDrift {
		out = append(out, scrapedMetric{
			Name:       "ad_recon_drift_micro_max",
			LabelsHash: "",
			Value:      maxDrift,
		})
	}
	return out, nil
}

func metricValue(m *dto.Metric) (float64, bool) {
	if m == nil {
		return 0, false
	}
	switch {
	case m.Gauge != nil:
		return m.Gauge.GetValue(), true
	case m.Counter != nil:
		return m.Counter.GetValue(), true
	case m.Untyped != nil:
		return m.Untyped.GetValue(), true
	default:
		return 0, false
	}
}

func labelsHash(labels []*dto.LabelPair) string {
	if len(labels) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(labels))
	for _, lp := range labels {
		if lp == nil || lp.Name == nil || lp.Value == nil {
			continue
		}
		pairs = append(pairs, *lp.Name+"="+*lp.Value)
	}
	if len(pairs) == 0 {
		return ""
	}
	sort.Strings(pairs)
	sum := sha256.Sum256([]byte(strings.Join(pairs, "|")))
	return hex.EncodeToString(sum[:8])
}

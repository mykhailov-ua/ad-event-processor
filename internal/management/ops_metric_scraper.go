package management

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	db "espx/internal/ingestion/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultOpsMetricScrapeInterval = 15 * time.Second
	defaultOpsMetricRetention      = 24 * time.Hour
	defaultOpsMetricScrapeTimeout  = 5 * time.Second
)

type OpsMetricScraper struct {
	svc      *Service
	pool     *pgxpool.Pool
	url      string
	client   *http.Client
	interval time.Duration
	retention time.Duration
	fetch    func(ctx context.Context, url string) ([]byte, string, error)
}

func NewOpsMetricScraper(svc *Service, scrapeURL string) *OpsMetricScraper {
	if scrapeURL == "" {
		scrapeURL = "http://127.0.0.1:8188/metrics"
	}
	client := &http.Client{Timeout: defaultOpsMetricScrapeTimeout}
	w := &OpsMetricScraper{
		svc:       svc,
		interval:  defaultOpsMetricScrapeInterval,
		retention: defaultOpsMetricRetention,
		client:    client,
		url:       scrapeURL,
	}
	if svc != nil {
		w.pool = svc.GetPool()
	}
	w.fetch = func(ctx context.Context, url string) ([]byte, string, error) {
		return fetchMetrics(ctx, client, url)
	}
	return w
}

func fetchMetrics(ctx context.Context, client *http.Client, url string) ([]byte, string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("metrics scrape status %d", resp.StatusCode)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func (s *Service) StartOpsMetricScraper(ctx context.Context, scrapeURL string) {
	if s == nil || s.GetPool() == nil {
		return
	}
	w := NewOpsMetricScraper(s, scrapeURL)
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
}

func (w *OpsMetricScraper) Start(ctx context.Context) {
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

func (w *OpsMetricScraper) tick(ctx context.Context, now time.Time) {
	run := func(runCtx context.Context) error {
		if err := w.scrapeAndStore(runCtx, now); err != nil {
			return err
		}
		return w.expireSamples(runCtx, now)
	}
	if w.svc != nil {
		if err := w.svc.withPgLow(ctx, run); err != nil {
			slog.Error("ops metric scraper tick failed", "error", err)
		}
		return
	}
	if err := run(ctx); err != nil {
		slog.Error("ops metric scraper tick failed", "error", err)
	}
}

func (w *OpsMetricScraper) scrapeAndStore(ctx context.Context, now time.Time) error {
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
	q := db.New(w.pool)
	ts := pgtype.Timestamptz{Time: now, Valid: true}
	for _, sample := range samples {
		if err := q.InsertOpsMetricSample(ctx, db.InsertOpsMetricSampleParams{
			Name:       sample.Name,
			LabelsHash: sample.LabelsHash,
			Ts:         ts,
			Value:      sample.Value,
		}); err != nil {
			return fmt.Errorf("insert metric sample name=%s: %w", sample.Name, err)
		}
	}
	return nil
}

func (w *OpsMetricScraper) expireSamples(ctx context.Context, now time.Time) error {
	cutoff := now.Add(-w.retention)
	q := db.New(w.pool)
	if _, err := q.DeleteExpiredOpsMetricSamples(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true}); err != nil {
		return fmt.Errorf("expire metric samples: %w", err)
	}
	return nil
}

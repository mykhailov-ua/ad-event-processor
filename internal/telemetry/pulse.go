package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultPulseTimeout = 5 * time.Second

type Metadata struct {
	DeploymentID  string `json:"deployment_id"`
	BinaryVersion string `json:"binary_version"`
	DCRegion      string `json:"dc_region,omitempty"`
}

type MetadataFunc func(ctx context.Context) (Metadata, error)

type PulsePayload struct {
	SchemaVersion  int    `json:"schema_version"`
	DeploymentID   string `json:"deployment_id"`
	BinaryVersion  string `json:"binary_version"`
	WindowSec      int    `json:"window_sec"`
	AcceptedEvents uint64 `json:"accepted_events"`
	RejectedEvents uint64 `json:"rejected_events"`
	PeakRPS        uint64 `json:"peak_rps"`
	DCRegion       string `json:"dc_region,omitempty"`
}

type Config struct {
	OptIn            bool
	URL              string
	LicenseServerURL string
	Interval         time.Duration
	HTTPTimeout      time.Duration
	WindowSec        int
	HTTPClient       *http.Client
	Metadata         MetadataFunc
}

type Worker struct {
	cfg Config
}

func NewWorker(cfg Config) *Worker {
	if cfg.Interval <= 0 {
		cfg.Interval = time.Hour
	}
	if cfg.WindowSec <= 0 {
		cfg.WindowSec = int(cfg.Interval / time.Second)
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = defaultPulseTimeout
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: cfg.HTTPTimeout}
	}
	return &Worker{cfg: cfg}
}

func (w *Worker) Enabled() bool {
	if w == nil {
		return false
	}
	return w.cfg.OptIn && strings.TrimSpace(w.cfg.URL) != ""
}

func (w *Worker) Start(ctx context.Context) {
	if !w.Enabled() {
		return
	}
	if err := w.validateEndpoints(); err != nil {
		slog.Error("product telemetry disabled", "error", err)
		return
	}
	slog.Info("product telemetry pulse enabled", "url", w.cfg.URL, "interval", w.cfg.Interval)

	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	w.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	snap := SnapshotAndReset()
	meta, err := w.metadata(ctx)
	if err != nil {
		slog.Warn("telemetry pulse metadata unavailable", "error", err)
		return
	}
	payload := PulsePayload{
		SchemaVersion:  1,
		DeploymentID:   meta.DeploymentID,
		BinaryVersion:  meta.BinaryVersion,
		WindowSec:      w.cfg.WindowSec,
		AcceptedEvents: snap.AcceptedEvents,
		RejectedEvents: snap.RejectedEvents,
		PeakRPS:        snap.PeakRPS,
		DCRegion:       meta.DCRegion,
	}
	if err := w.upload(ctx, payload); err != nil {
		slog.Warn("telemetry pulse upload failed", "error", err)
	}
}

func (w *Worker) metadata(ctx context.Context) (Metadata, error) {
	if w.cfg.Metadata == nil {
		return Metadata{}, errors.New("telemetry metadata provider not configured")
	}
	return w.cfg.Metadata(ctx)
}

func (w *Worker) upload(ctx context.Context, payload PulsePayload) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := ValidatePayloadJSON(raw); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.cfg.URL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := w.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telemetry pulse status %d", resp.StatusCode)
	}
	if elapsed := time.Since(start); elapsed > defaultPulseTimeout {
		slog.Warn("telemetry pulse slow upload", "duration", elapsed)
	}
	return nil
}

func (w *Worker) validateEndpoints() error {
	pulseHost, err := endpointHost(w.cfg.URL)
	if err != nil {
		return err
	}
	licenseHost, err := endpointHost(w.cfg.LicenseServerURL)
	if err != nil || pulseHost == "" || licenseHost == "" {
		return nil
	}
	if pulseHost == licenseHost {
		return errors.New("ESPX_TELEMETRY_URL must differ from ESPX_LICENSE_SERVER host")
	}
	return nil
}

func endpointHost(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("telemetry endpoint: %w", err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("telemetry endpoint missing host")
	}
	return strings.ToLower(u.Scheme + "://" + u.Host), nil
}

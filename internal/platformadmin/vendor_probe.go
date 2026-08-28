package platformadmin

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

type Probe interface {
	Name() string
	Probe(ctx context.Context) error
}

type Observer interface {
	ObserveProbe(vendor string, success bool, latency time.Duration)
	ObserveProbeError(vendor string)
}

type Registry struct {
	probes []Probe
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Register(p Probe) {
	if r == nil || p == nil {
		return
	}
	r.probes = append(r.probes, p)
}

func (r *Registry) Probes() []Probe {
	if r == nil {
		return nil
	}
	out := make([]Probe, len(r.probes))
	copy(out, r.probes)
	return out
}

type WorkerConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

type Worker struct {
	reg      *Registry
	cfg      WorkerConfig
	observer Observer
}

func NewWorker(reg *Registry, cfg WorkerConfig, observer Observer) *Worker {
	if cfg.Interval <= 0 {
		cfg.Interval = 60 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &Worker{reg: reg, cfg: cfg, observer: observer}
}

func (w *Worker) Start(ctx context.Context) {
	if w == nil || w.reg == nil || len(w.reg.probes) == 0 {
		return
	}
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	w.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *Worker) runOnce(parent context.Context) {
	probes := w.reg.Probes()
	var wg sync.WaitGroup
	for _, p := range probes {
		probe := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.runProbe(parent, probe)
		}()
	}
	wg.Wait()
}

func (w *Worker) runProbe(parent context.Context, p Probe) {
	ctx, cancel := context.WithTimeout(parent, w.cfg.Timeout)
	defer cancel()

	start := time.Now()
	err := p.Probe(ctx)
	latency := time.Since(start)
	success := err == nil

	if err != nil {
		slog.Warn("vendor probe failed", "vendor", p.Name(), "error", err, "latency_ms", latency.Milliseconds())
		if w.observer != nil {
			w.observer.ObserveProbeError(p.Name())
		}
	}
	if w.observer != nil {
		w.observer.ObserveProbe(p.Name(), success, latency)
	}
}

var (
	stripeBalanceURL = "https://api.stripe.com/v1/balance"
	telegramAPIBase  = "https://api.telegram.org/bot"
)

type MaxMindProbe struct {
	DBPath string
}

func NewMaxMindProbe(dbPath string) *MaxMindProbe {
	return &MaxMindProbe{DBPath: dbPath}
}

func (p *MaxMindProbe) Name() string { return "maxmind" }

func (p *MaxMindProbe) Probe(ctx context.Context) error {
	if p == nil || p.DBPath == "" {
		return nil
	}
	info, err := os.Stat(p.DBPath)
	if err != nil {
		return fmt.Errorf("stat geoip db: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("geoip db empty")
	}
	return nil
}

type StripeProbe struct {
	SecretKey string
	client    *http.Client
}

func NewStripeProbe(secretKey string, client *http.Client) *StripeProbe {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &StripeProbe{SecretKey: secretKey, client: client}
}

func (p *StripeProbe) Name() string { return "stripe" }

func (p *StripeProbe) Probe(ctx context.Context) error {
	if p == nil || p.SecretKey == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, stripeBalanceURL, http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.SecretKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("stripe balance status %d", resp.StatusCode)
	}
	return nil
}

type TelegramProbe struct {
	BotToken string
	client   *http.Client
}

func NewTelegramProbe(botToken string, client *http.Client) *TelegramProbe {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &TelegramProbe{BotToken: botToken, client: client}
}

func (p *TelegramProbe) Name() string { return "telegram" }

func (p *TelegramProbe) Probe(ctx context.Context) error {
	if p == nil || p.BotToken == "" {
		return nil
	}
	url := telegramAPIBase + p.BotToken + "/getMe"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram getMe status %d", resp.StatusCode)
	}
	return nil
}

type SMTPProbe struct {
	Host string
	Port string
}

func NewSMTPProbe(host, port string) *SMTPProbe {
	if port == "" {
		port = "587"
	}
	return &SMTPProbe{Host: host, Port: port}
}

func (p *SMTPProbe) Name() string { return "smtp" }

func (p *SMTPProbe) Probe(ctx context.Context) error {
	if p == nil || p.Host == "" {
		return nil
	}
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(p.Host, p.Port))
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

type Options struct {
	GeoIPDBPath      string
	StripeSecretKey  string
	TelegramBotToken string
	SMTPHost         string
	SMTPPort         string
}

func RegistryFromOptions(opts Options) *Registry {
	reg := NewRegistry()
	reg.Register(NewMaxMindProbe(opts.GeoIPDBPath))
	reg.Register(NewStripeProbe(opts.StripeSecretKey, nil))
	reg.Register(NewTelegramProbe(opts.TelegramBotToken, nil))
	reg.Register(NewSMTPProbe(opts.SMTPHost, opts.SMTPPort))
	return reg
}

package vendorprobe

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

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

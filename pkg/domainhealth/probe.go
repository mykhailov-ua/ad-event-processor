// Package domainhealth implements domainhealth support for BidShard.
package domainhealth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/branding"
	"github.com/bidshard/ad-event-processor/pkg/platformconfig"
)

const (
	HealthHealthy  = "healthy"
	HealthDegraded = "degraded"
	HealthDown     = "down"
	HealthUnknown  = "unknown"

	SSLValid    = "valid"
	SSLExpiring = "expiring"
	SSLExpired  = "expired"
	SSLMissing  = "missing"
	SSLUnknown  = "unknown"

	roleTracking = "tracking"
	roleAdmin    = "admin"

	probeTimeout       = 10 * time.Second
	degradedLatencyMs  = 2000
	sslExpiringHorizon = 14 * 24 * time.Hour
)

type Result struct {
	Hostname       string
	Role           string
	HealthStatus   string
	SSLStatus      string
	SSLNotAfter    *time.Time
	HTTPStatus     int
	ProbeLatencyMs int64
	ProbeDetail    string
}

func ProbePath(role string) string {
	if role == roleAdmin {
		return "/healthz"
	}
	return "/health"
}

func Probe(ctx context.Context, hostname, role string) Result {
	host := platformconfig.ResolveHost(hostname)
	res := Result{
		Hostname:     host,
		Role:         role,
		HealthStatus: HealthUnknown,
		SSLStatus:    SSLUnknown,
	}
	if host == "" {
		res.HealthStatus = HealthDown
		res.ProbeDetail = "hostname empty"
		return res
	}

	start := time.Now()
	path := ProbePath(role)
	url := "https://" + host + path

	tlsInfo, tlsErr := probeTLS(ctx, host)
	if tlsErr == nil && tlsInfo.NotAfter != nil {
		res.SSLNotAfter = tlsInfo.NotAfter
		res.SSLStatus = classifySSL(*tlsInfo.NotAfter, time.Now())
	} else if tlsErr != nil {
		res.SSLStatus = SSLMissing
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, http.NoBody)
	if err != nil {
		res.HealthStatus = HealthDown
		res.ProbeDetail = err.Error()
		res.ProbeLatencyMs = time.Since(start).Milliseconds()
		return res
	}
	req.Header.Set("User-Agent", branding.HTTPUserAgent("DomainHealth"))

	client := &http.Client{
		Timeout: probeTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	res.ProbeLatencyMs = latency
	if err != nil {
		res.HealthStatus = HealthDown
		res.ProbeDetail = err.Error()
		if res.SSLStatus == SSLUnknown {
			res.SSLStatus = SSLMissing
		}
		return res
	}
	defer func() { _ = resp.Body.Close() }()
	res.HTTPStatus = resp.StatusCode

	res.HealthStatus, res.ProbeDetail = classifyHTTP(resp.StatusCode, latency, res.SSLStatus)
	if res.SSLStatus == SSLExpiring && res.HealthStatus == HealthHealthy {
		res.HealthStatus = HealthDegraded
		if res.ProbeDetail == "" {
			res.ProbeDetail = "ssl expiring soon"
		} else {
			res.ProbeDetail += "; ssl expiring soon"
		}
	}
	return res
}

type tlsProbeResult struct {
	NotAfter *time.Time
}

func probeTLS(ctx context.Context, host string) (tlsProbeResult, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, "443"), &tls.Config{
		ServerName: host,
	})
	if err != nil {
		return tlsProbeResult{}, err
	}
	defer func() { _ = conn.Close() }()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return tlsProbeResult{}, fmt.Errorf("no peer certificate")
	}
	notAfter := certs[0].NotAfter.UTC()
	return tlsProbeResult{NotAfter: &notAfter}, nil
}

func classifySSL(notAfter time.Time, now time.Time) string {
	remaining := notAfter.Sub(now)
	if remaining <= 0 {
		return SSLExpired
	}
	if remaining <= sslExpiringHorizon {
		return SSLExpiring
	}
	return SSLValid
}

func classifyHTTP(status int, latencyMs int64, sslStatus string) (health, detail string) {
	if status >= 200 && status < 400 {
		health = HealthHealthy
		detail = fmt.Sprintf("http %d", status)
		if latencyMs >= degradedLatencyMs {
			health = HealthDegraded
			detail = fmt.Sprintf("http %d slow %dms", status, latencyMs)
		}
		return health, detail
	}
	if status > 0 {
		return HealthDown, fmt.Sprintf("http %d", status)
	}
	if sslStatus == SSLExpired || sslStatus == SSLMissing {
		return HealthDown, "tls unavailable"
	}
	return HealthDown, "probe failed"
}

func NormalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case roleAdmin:
		return roleAdmin
	case "custom":
		return "custom"
	default:
		return roleTracking
	}
}

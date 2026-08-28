package opsadmin

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type EdgeMetricsPanelDTO struct {
	UpdatedAt      string            `json:"updated_at,omitempty"`
	IngressH1      uint64            `json:"ingress_h1"`
	IngressH2      uint64            `json:"ingress_h2"`
	IngressH3      uint64            `json:"ingress_h3"`
	BodyStream     uint64            `json:"body_stream"`
	BodyPeek       uint64            `json:"body_peek"`
	BodyRead       uint64            `json:"body_read"`
	Blocked        map[string]uint64 `json:"blocked"`
	TarpitTotal    uint64            `json:"tarpit_total"`
	BlacklistStale uint64            `json:"blacklist_stale"`
}

const (
	edgeMetricsTimeout   = 3 * time.Second
	edgeMetricsMaxBody   = 4 << 20
	edgeMetricsNameParts = 2
)

var edgeMetricsClient = &http.Client{Timeout: edgeMetricsTimeout}

func FetchEdgeMetrics(ctx context.Context) (EdgeMetricsPanelDTO, error) {
	rawURL := strings.TrimSpace(os.Getenv("EDGE_METRICS_URL"))
	if rawURL == "" {
		rawURL = "http://127.0.0.1:8180/metrics/edge"
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return EdgeMetricsPanelDTO{}, fmt.Errorf("edge metrics url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return EdgeMetricsPanelDTO{}, fmt.Errorf("edge metrics url scheme not allowed")
	}
	if host := parsed.Hostname(); host != "" && host != "127.0.0.1" && host != "localhost" {
		return EdgeMetricsPanelDTO{}, fmt.Errorf("edge metrics host not allowed")
	}

	reqCtx, cancel := context.WithTimeout(ctx, edgeMetricsTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return EdgeMetricsPanelDTO{}, err
	}
	res, err := edgeMetricsClient.Do(req)
	if err != nil {
		return EdgeMetricsPanelDTO{}, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return EdgeMetricsPanelDTO{}, fmt.Errorf("edge metrics status %d", res.StatusCode)
	}
	limited := io.LimitReader(res.Body, edgeMetricsMaxBody)
	return parseEdgePrometheus(limited), nil
}

func parseEdgePrometheus(r io.Reader) EdgeMetricsPanelDTO {
	out := EdgeMetricsPanelDTO{
		Blocked:   map[string]uint64{},
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, val, ok := parsePrometheusSample(line)
		if !ok {
			continue
		}
		switch {
		case edgeMetricMatch(name, "ad_event_processor_edge_ingress_protocol_h1_total"):
			out.IngressH1 = val
		case edgeMetricMatch(name, "ad_event_processor_edge_ingress_protocol_h2_total"):
			out.IngressH2 = val
		case edgeMetricMatch(name, "ad_event_processor_edge_ingress_protocol_h3_total"):
			out.IngressH3 = val
		case edgeMetricMatch(name, "ad_event_processor_edge_body_stream_total"):
			out.BodyStream = val
		case edgeMetricMatch(name, "ad_event_processor_edge_body_peek_total"):
			out.BodyPeek = val
		case edgeMetricMatch(name, "ad_event_processor_edge_body_read_total"):
			out.BodyRead = val
		case edgeMetricMatch(name, "ad_event_processor_edge_blocked_ip_total"):
			out.Blocked["ip_blacklist"] = val
		case edgeMetricMatch(name, "ad_event_processor_edge_blocked_campaign_rl_total"):
			out.Blocked["campaign_rl"] = val
		case edgeMetricMatch(name, "ad_event_processor_edge_blocked_fraud_tier_total"):
			out.Blocked["fraud_tier"] = val
		case edgeMetricMatch(name, "ad_event_processor_edge_circuit_reject_total"):
			out.Blocked["circuit_breaker"] = val
		case edgeMetricMatch(name, "ad_event_processor_edge_parse_oversize_total"):
			out.Blocked["parse_oversize"] = val
		case edgeMetricMatch(name, "ad_event_processor_edge_chunked_reject_total"):
			out.Blocked["chunked_reject"] = val
		case edgeMetricMatch(name, "ad_event_processor_edge_tarpit_total"):
			out.TarpitTotal = val
		case edgeMetricMatch(name, "ad_event_processor_edge_blacklist_stale_total"):
			out.BlacklistStale = val
		}
	}
	if err := sc.Err(); err != nil {
		return out
	}
	return out
}

func parsePrometheusSample(line string) (string, uint64, bool) {
	parts := strings.Fields(line)
	if len(parts) < edgeMetricsNameParts {
		return "", 0, false
	}
	name := parts[0]
	if i := strings.IndexByte(name, '{'); i >= 0 {
		name = name[:i]
	}
	val, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return "", 0, false
	}
	return name, val, true
}

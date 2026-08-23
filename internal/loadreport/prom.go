package loadreport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type promClient struct {
	baseURL string
	client  *http.Client
}

type promVectorRow struct {
	Value  string
	Labels string
}

func newPromClient(promURL string) *promClient {
	return &promClient{
		baseURL: strings.TrimRight(promURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *promClient) scalar(query string) string {
	v, err := c.queryScalar(query)
	if err != nil || v == "" {
		return "na"
	}
	return v
}

func (c *promClient) scalarOrZero(query string) string {
	v, err := c.queryScalar(query)
	if err != nil || v == "" {
		return "0"
	}
	return v
}

func (c *promClient) queryScalar(query string) (string, error) {
	body, err := c.query(query)
	if err != nil {
		return "", err
	}
	var resp promQueryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	results := resp.Data.Result
	if len(results) == 0 {
		return "", nil
	}
	val, ok := results[0].Value[1].(string)
	if !ok {
		return "", nil
	}
	return val, nil
}

func (c *promClient) vector(query string) ([]promVectorRow, error) {
	body, err := c.query(query)
	if err != nil {
		return nil, err
	}
	var resp promQueryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	var rows []promVectorRow
	for _, r := range resp.Data.Result {
		val, _ := r.Value[1].(string)
		var labels []string
		for k, v := range r.Metric {
			if k == "__name__" {
				continue
			}
			labels = append(labels, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(labels)
		rows = append(rows, promVectorRow{Value: val, Labels: strings.Join(labels, ",")})
	}
	return rows, nil
}

func (c *promClient) query(query string) ([]byte, error) {
	u, err := url.Parse(c.baseURL + "/api/v1/query")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("query", query)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(resp.Body)
}

type promQueryResponse struct {
	Data struct {
		Result []struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type statusHistogramFile struct {
	ByStatus  map[string]int64 `json:"by_status"`
	Histogram []struct {
		Status string `json:"status"`
		Error  string `json:"error"`
		Count  int64  `json:"count"`
	} `json:"histogram"`
}

func WritePromReports(outDir, promURL string) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	prom := newPromClient(promURL)
	reportPath := filepath.Join(outDir, "bottleneck-report.md")
	var b strings.Builder
	writeBottleneckReport(&b, outDir, promURL, prom)
	if err := os.WriteFile(reportPath, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return reportPath, nil
}

func writeBottleneckReport(b *strings.Builder, outDir, promURL string, prom *promClient) {
	fmt.Fprintf(b, "# Dirty Load Bottleneck Report\n\n")
	fmt.Fprintf(b, "Generated: %s\n", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(b, "Prometheus: %s\n", promURL)
	b.WriteString("Grafana: http://127.0.0.1:3100\n\n")

	b.WriteString("## Ingress (tracker gnet)\n\n")
	b.WriteString("| Metric | Value |\n")
	b.WriteString("|--------|-------|\n")
	fmt.Fprintf(b, "| Tracker RPS | %s |\n", prom.scalar(`sum(rate(ad_http_request_duration_seconds_count{job="tracker"}[5m]))`))
	fmt.Fprintf(b, "| Tracker p95 (ms) | %s |\n", prom.scalar(`histogram_quantile(0.95, sum(rate(ad_http_request_duration_seconds_bucket{job="tracker"}[5m])) by (le)) * 1000`))
	fmt.Fprintf(b, "| Tracker p99 (ms) | %s |\n", prom.scalar(`histogram_quantile(0.99, sum(rate(ad_http_request_duration_seconds_bucket{job="tracker"}[5m])) by (le)) * 1000`))
	fmt.Fprintf(b, "| OpenRTB exchange RPS | %s |\n", prom.scalar(`sum(rate(ad_rtb_exchange_request_total{job="tracker"}[5m]))`))
	fmt.Fprintf(b, "| OpenRTB exchange p99 (ms) | %s |\n", prom.scalar(`histogram_quantile(0.99, sum(rate(ad_rtb_exchange_duration_seconds_bucket{job="tracker"}[5m])) by (le)) * 1000`))
	fmt.Fprintf(b, "| OpenRTB validate errors/s | %s |\n", prom.scalar(`sum(rate(ad_rtb_exchange_validate_errors_total{job="tracker"}[5m]))`))
	fmt.Fprintf(b, "| gnet active connections | %s |\n", prom.scalar(`sum(ad_gnet_active_connections{job="tracker"})`))
	fmt.Fprintf(b, "| Worker pool rejects/s | %s |\n", prom.scalar(`sum(rate(ad_worker_pool_reject_total{job="tracker"}[5m]))`))
	fmt.Fprintf(b, "| HTTP parse errors/s | %s |\n", prom.scalar(`sum(rate(ad_http_parse_errors_total{job="tracker"}[5m]))`))
	b.WriteString("\n")

	b.WriteString("## Redis Lua (user/kernel boundary: TCP + epoll + write)\n\n")
	b.WriteString("| Shard | Lua p99 (ms) | ops/s | NOSCRIPT/s |\n")
	b.WriteString("|-------|--------------|-------|------------|\n")
	shards, _ := prom.vector(`sum(rate(ad_redis_ops_total{job="tracker"}[5m])) by (shard)`)
	for _, row := range shards {
		if row.Labels == "" {
			continue
		}
		shard := strings.TrimPrefix(row.Labels, "shard=")
		p99 := prom.scalar(fmt.Sprintf(`histogram_quantile(0.99, sum(rate(ad_redis_lua_duration_seconds_bucket{job="tracker",shard=%q}[5m])) by (le)) * 1000`, shard))
		ops := prom.scalar(fmt.Sprintf(`sum(rate(ad_redis_ops_total{job="tracker",shard=%q}[5m]))`, shard))
		noscript := prom.scalar(fmt.Sprintf(`sum(rate(ad_redis_lua_noscript_total{job="tracker",shard=%q}[5m]))`, shard))
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", shard, p99, ops, noscript)
	}
	b.WriteString("\n")

	b.WriteString("## Filter / fraud path\n\n")
	fmt.Fprintf(b, "- Fraud stream drops/s: %s\n", prom.scalar(`sum(rate(ad_fraud_stream_drop_total{job="tracker"}[5m]))`))
	fmt.Fprintf(b, "- Events dropped (Redis ingest)/s: %s\n", prom.scalar(`sum(rate(ad_events_dropped_total{job="tracker"}[5m]))`))
	fmt.Fprintf(b, "- Fraud tier block/s: %s\n", prom.scalar(`sum(rate(ad_filter_blocked_total{reason="fraud"}[5m]))`))
	fmt.Fprintf(b, "- Budget cache miss PG/s: %s\n", prom.scalar(`sum(rate(ad_budget_cache_miss_pg_total{job="tracker"}[5m]))`))
	b.WriteString("\n")
	b.WriteString("### Filter blocks by reason\n")
	b.WriteString("```\n")
	reasons, _ := prom.vector(`sum(rate(ad_filter_blocked_total{job="tracker"}[5m])) by (reason)`)
	sort.Slice(reasons, func(i, j int) bool {
		vi, _ := strconv.ParseFloat(reasons[i].Value, 64)
		vj, _ := strconv.ParseFloat(reasons[j].Value, 64)
		return vi > vj
	})
	limit := 20
	if len(reasons) < limit {
		limit = len(reasons)
	}
	for _, row := range reasons[:limit] {
		fmt.Fprintf(b, "%s\t%s\n", row.Value, row.Labels)
	}
	b.WriteString("```\n\n")

	b.WriteString("## Cold path: pgx (Postgres) + clickhouse-go\n\n")
	b.WriteString("| Store | p99 batch write (ms) | errors/s |\n")
	b.WriteString("|-------|---------------------|----------|\n")
	for _, typ := range []string{"postgres", "clickhouse"} {
		p99 := prom.scalar(fmt.Sprintf(`histogram_quantile(0.99, sum(rate(ad_db_write_duration_seconds_bucket{job="processor",type=%q}[5m])) by (le)) * 1000`, typ))
		errRate := prom.scalar(fmt.Sprintf(`sum(rate(ad_db_write_errors_total{job="processor",type=%q}[5m]))`, typ))
		if errRate == "na" {
			errRate = "0"
		}
		fmt.Fprintf(b, "| %s | %s | %s |\n", typ, p99, errRate)
	}
	b.WriteString("\n")
	fmt.Fprintf(b, "- Tracker stream ingest/s: %s\n", prom.scalar(`sum(rate(ad_events_processed_total{job=~"tracker.*"}[5m]))`))
	fmt.Fprintf(b, "- Processor PG write errors/s: %s\n", prom.scalar(`sum(rate(ad_db_write_errors_total{job="processor",type="postgres"}[5m]))`))
	fmt.Fprintf(b, "- Processor CH write errors/s: %s\n", prom.scalar(`sum(rate(ad_db_write_errors_total{job="processor",type="clickhouse"}[5m]))`))
	fmt.Fprintf(b, "- Processor stream lag (sum xlen): %s\n", prom.scalar(`sum(ad_processor_stream_xlen{job="processor"})`))
	fmt.Fprintf(b, "- DLQ size: %s\n", prom.scalar(`sum(ad_dlq_size_total{job="processor"})`))
	b.WriteString("\n")

	b.WriteString("## Edge (nginx OpenResty)\n\n")
	fmt.Fprintf(b, "- Phase1 pass/s: %s\n", prom.scalar(`sum(rate(ad_event_processor_edge_phase1_pass_total[5m]))`))
	fmt.Fprintf(b, "- Circuit reject/s: %s\n", prom.scalar(`sum(rate(ad_event_processor_edge_circuit_reject_total[5m]))`))
	fmt.Fprintf(b, "- Blocked IP/s: %s\n", prom.scalar(`sum(rate(ad_event_processor_edge_blocked_ip_total[5m]))`))
	b.WriteString("\n")

	b.WriteString("## File descriptors & syscalls\n\n")
	writeStraceSection(b, outDir)
	b.WriteString("\n")

	b.WriteString("## Interpretation hints\n\n")
	b.WriteString("1. **Redis Lua p99 > 15ms** — kernel TCP/epoll or Redis single-threaded shard saturation; check `ad_redis_ops_total` per shard.\n")
	b.WriteString("2. **clickhouse p99 >> postgres** — async_insert batching or LSM merge pressure; check CH `system.parts` and processor `ad_db_write_errors{type=clickhouse}`.\n")
	b.WriteString("3. **pgx p99 spikes** — Postgres WAL/fsync or pool exhaustion (`DB_PROCESSOR_MAX_CONNS`); strace shows `write`/`fsync`/`epoll_wait` dominance.\n")
	b.WriteString("4. **gnet connections near ulimit** — raise `worker_rlimit_nofile` / container ulimits; loadgen keep-alive reduces FD churn.\n")
	b.WriteString("5. **fraud_stream_drop > 0** — fraud ring (4096) overflow; hot path lossy by design under malformed traffic.\n")
	b.WriteString("6. **worker_pool_reject** — pinned worker queue full; ingestion exceeds parse+filter capacity.\n")

	hist, histLabel := readStatusHistogram(outDir)
	if hist != nil {
		writeStatusHistogramSection(b, hist, histLabel)
	}

	write5xxSection(b, outDir, promURL, prom, hist)

	bpfReport := filepath.Join(outDir, "bpf-report.md")
	if bpfData, err := os.ReadFile(bpfReport); err == nil {
		b.WriteString("\n## BPF probe (dev)\n\n")
		b.WriteString("Kernel/session detail: [bpf-report.md](bpf-report.md).\n")
		for _, line := range strings.Split(string(bpfData), "\n") {
			if strings.Contains(line, "loadgen share of tracked on-CPU") {
				fmt.Fprintf(b, "- %s\n", strings.TrimSpace(line))
				break
			}
		}
	}
}

func writeStraceSection(b *strings.Builder, outDir string) {
	matches, _ := filepath.Glob(filepath.Join(outDir, "*-strace-*.txt"))
	if len(matches) > 0 {
		fmt.Fprintf(b, "Strace summaries captured under `%s`. Top syscalls per service:\n", outDir)
		b.WriteString("```\n")
		for _, f := range matches {
			fmt.Fprintf(b, "=== %s ===\n", filepath.Base(f))
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			lines := strings.Split(string(data), "\n")
			shown := 0
			for _, line := range lines {
				if strings.Contains(line, "%") {
					fmt.Fprintf(b, "%s\n", line)
					shown++
					if shown >= 8 {
						break
					}
				}
			}
		}
		b.WriteString("```\n")
	} else {
		b.WriteString("_No strace samples. Re-run with snapshot_runtime.sh during load._\n")
	}

	reportMatches, _ := filepath.Glob(filepath.Join(outDir, "ad-event-processor-*.txt"))
	if len(reportMatches) > 0 {
		b.WriteString("\nFD counts:\n")
		b.WriteString("```\n")
		shown := 0
		for _, f := range reportMatches {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			lines := strings.Split(string(data), "\n")
			for i, line := range lines {
				if strings.Contains(line, "fd_count") || strings.Contains(line, "## fd_count") {
					if i+1 < len(lines) {
						fmt.Fprintf(b, "%s %s\n", strings.TrimSpace(line), strings.TrimSpace(lines[i+1]))
						shown++
					}
					if shown >= 10 {
						break
					}
				}
			}
			if shown >= 10 {
				break
			}
		}
		b.WriteString("```\n")
	}
}

func readStatusHistogram(outDir string) (hist *statusHistogramFile, label string) {
	path := filepath.Join(outDir, "status-histogram.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, ""
	}
	var h statusHistogramFile
	if json.Unmarshal(data, &h) != nil {
		return nil, ""
	}
	return &h, "loadgen"
}

func writeStatusHistogramSection(b *strings.Builder, hist *statusHistogramFile, label string) {
	fmt.Fprintf(b, "\n## %s HTTP status histogram\n\n", label)
	total := int64(0)
	for _, v := range hist.ByStatus {
		total += v
	}
	if total == 0 {
		total = 1
	}
	b.WriteString("| status | count | % |\n")
	b.WriteString("|--------|-------|---|\n")
	type kv struct {
		status string
		count  int64
	}
	var rows []kv
	for status, count := range hist.ByStatus {
		rows = append(rows, kv{status, count})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].count > rows[j].count })
	for _, row := range rows {
		pct := 100.0 * float64(row.count) / float64(total)
		mark := ""
		if n, err := strconv.Atoi(row.status); err == nil && n >= 500 {
			mark = " **5xx**"
		}
		fmt.Fprintf(b, "| %s | %d | %.1f%% |%s\n", row.status, row.count, pct, mark)
	}
	b.WriteString("\nTop status×error buckets:\n")
	limit := 12
	if len(hist.Histogram) < limit {
		limit = len(hist.Histogram)
	}
	for _, row := range hist.Histogram[:limit] {
		fmt.Fprintf(b, "- status=%s error=%s: %d\n", row.Status, row.Error, row.Count)
	}
	var five int64
	for status, count := range hist.ByStatus {
		if n, err := strconv.Atoi(status); err == nil && n >= 500 {
			five += count
		}
	}
	fmt.Fprintf(b, "\n_%s 5xx total: %d (%.2f%% of classified responses)_\n", label, five, 100*float64(five)/float64(total))
}

func write5xxSection(b *strings.Builder, outDir, promURL string, prom *promClient, hist *statusHistogramFile) {
	b.WriteString("\n## 5xx reconciliation\n\n")
	fmt.Fprintf(b, "Session: `%s`\n\n", outDir)

	if hist != nil {
		b.WriteString("### Client status histogram (5xx detail)\n\n")
		total := int64(0)
		for _, v := range hist.ByStatus {
			total += v
		}
		if total == 0 {
			total = 1
		}
		b.WriteString("| status | count | % |\n")
		b.WriteString("|--------|-------|---|\n")
		type kv struct {
			status string
			count  int64
		}
		var rows []kv
		for status, count := range hist.ByStatus {
			rows = append(rows, kv{status, count})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].count > rows[j].count })
		var five int64
		for _, row := range rows {
			pct := 100.0 * float64(row.count) / float64(total)
			flag := ""
			if n, err := strconv.Atoi(row.status); err == nil && n >= 500 {
				flag = " **5xx**"
				five += row.count
			}
			fmt.Fprintf(b, "| %s | %d | %.1f%% |%s\n", row.status, row.count, pct, flag)
		}
		zero := hist.ByStatus["0"]
		fmt.Fprintf(b, "\n- Client status\u2265500: **%d** (%.2f%%)\n", five, 100*float64(five)/float64(total))
		fmt.Fprintf(b, "- Client status=0 (transport): **%d** (%.2f%%)\n", zero, 100*float64(zero)/float64(total))
		b.WriteString("\nTop buckets (status × error):\n")
		limit := 15
		if len(hist.Histogram) < limit {
			limit = len(hist.Histogram)
		}
		for _, row := range hist.Histogram[:limit] {
			fmt.Fprintf(b, "- `%s` / `%s`: %d\n", row.Status, row.Error, row.Count)
		}
	} else {
		b.WriteString("_No status-histogram.json in session dir._\n")
	}

	b.WriteString("\n### Tracker Prometheus (`ad_http_requests_total`, increase 15m)\n\n")
	b.WriteString("| status | POST /track count |\n")
	b.WriteString("|--------|-------------------|\n")
	trackerFive := writeTrackerStatusTable(b, prom)
	b.WriteString("\n### Correlation\n\n")

	clientFive := "na"
	if hist != nil {
		var five int64
		for status, count := range hist.ByStatus {
			if n, err := strconv.Atoi(status); err == nil && n >= 500 {
				five += count
			}
		}
		clientFive = strconv.FormatInt(five, 10)
	}
	trackerFiveStr := prom.scalarOrZero(`sum(increase(ad_http_requests_total{job="tracker",path="/track",status=~"5.."}[15m]))`)
	fmt.Fprintf(b, "- Client status\u2265500: **%s**\n", clientFive)
	fmt.Fprintf(b, "- Tracker `ad_http_requests_total` 5xx: **%s**\n", trackerFiveStr)
	if clientFive != "na" && clientFive != "0" && trackerFiveStr == "0" {
		b.WriteString("\n> **Gap:** client sees 5xx but tracker hot-path counters show none.\n")
		b.WriteString("> Dirty mix sends ~15% of requests to `EDGE_URL` (nginx, default :8180) — edge circuit breaker returns **503** under pressure.\n")
		b.WriteString("> Compare direct tracker buckets (400/404/413) with `status=0` transport errors on keep-alive.\n")
	}
	_ = trackerFive
	_ = promURL

	b.WriteString("\n### Related metrics\n\n")
	fmt.Fprintf(b, "- worker_pool_reject increase: %s\n", prom.scalarOrZero(`sum(increase(ad_worker_pool_reject_total{job="tracker"}[15m]))`))
	fmt.Fprintf(b, "- parse payload_too_large increase: %s\n", prom.scalarOrZero(`sum(increase(ad_http_parse_errors_total{job="tracker",error_type="payload_too_large"}[15m]))`))
	fmt.Fprintf(b, "- filter_engine failures increase: %s\n", prom.scalarOrZero(`sum(increase(ad_filter_internal_errors_total{job="tracker",kind="filter_engine"}[15m]))`))
}

func writeTrackerStatusTable(b *strings.Builder, prom *promClient) float64 {
	rows, err := prom.vector(`sum(increase(ad_http_requests_total{job="tracker",path="/track"}[15m])) by (status)`)
	if err != nil {
		fmt.Fprintf(b, "_Prometheus query failed: %v_\n", err)
		return 0
	}
	type statusRow struct {
		val    float64
		status string
	}
	var parsed []statusRow
	for _, r := range rows {
		v, err := strconv.ParseFloat(r.Value, 64)
		if err != nil || v < 0.5 {
			continue
		}
		status := ""
		for _, part := range strings.Split(r.Labels, ",") {
			if strings.HasPrefix(part, "status=") {
				status = strings.TrimPrefix(part, "status=")
			}
		}
		parsed = append(parsed, statusRow{val: v, status: status})
	}
	sort.Slice(parsed, func(i, j int) bool { return parsed[i].val > parsed[j].val })
	var five float64
	for _, row := range parsed {
		mark := ""
		if n, err := strconv.Atoi(row.status); err == nil && n >= 500 {
			five += row.val
			mark = " **5xx**"
		}
		fmt.Fprintf(b, "| %s | %.0f |%s\n", row.status, row.val, mark)
	}
	fmt.Fprintf(b, "\n- tracker instrumented 5xx: **%.0f**\n", five)
	return five
}

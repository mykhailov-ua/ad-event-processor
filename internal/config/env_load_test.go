package config

import (
	"strings"
	"testing"
	"time"
)

func TestTrimCommaList_dropsEmpty(t *testing.T) {
	got := trimCommaList(" a , ,b, ")
	want := []string{"a", "b"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%q want %q", i, got[i], want[i])
		}
	}
}

func TestResolveRedisMasterNames_defaults(t *testing.T) {
	cfg := &Config{RedisAddrs: []string{"h0:6379", "h1:6379", "h2:6379"}}
	names := cfg.ResolveRedisMasterNames()
	want := []string{"ad-event-processor-shard-0", "ad-event-processor-shard-1", "ad-event-processor-shard-2"}
	if len(names) != len(want) {
		t.Fatalf("len=%d want %d", len(names), len(want))
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names[%d]=%q want %q", i, names[i], want[i])
		}
	}
}

func TestResolveRedisMasterNames_explicit(t *testing.T) {
	cfg := &Config{
		RedisAddrs:       []string{"h0:6379", "h1:6379"},
		RedisMasterNames: []string{"custom-a", "custom-b"},
	}
	names := cfg.ResolveRedisMasterNames()
	if names[0] != "custom-a" || names[1] != "custom-b" {
		t.Fatalf("unexpected names: %v", names)
	}
}

func TestRedisSentinelEnabled(t *testing.T) {
	if (&Config{}).RedisSentinelEnabled() {
		t.Fatal("expected disabled with empty sentinel addrs")
	}
	if !(&Config{RedisSentinelAddrs: []string{"127.0.0.1:26379"}}).RedisSentinelEnabled() {
		t.Fatal("expected enabled when sentinel addrs set")
	}
}

func TestLoad_productionRequiresExpectedShardCount(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("SERVER_PORT", "8181")
	t.Setenv("DB_DSN", "postgres://u:p@localhost/db?sslmode=disable")
	t.Setenv("REDIS_ADDRS", "127.0.0.1:6379")
	t.Setenv("TOKEN_SYMMETRIC_KEY", "01234567890123456789012345678901")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for single shard in production")
	}

	t.Setenv("REDIS_ADDRS", strings.Join([]string{
		"127.0.0.1:6479", "127.0.0.1:6480", "127.0.0.1:6481", "127.0.0.1:6482",
	}, ","))
	t.Setenv("FILTER_TIMEOUT_MS", "100")
	_, err = Load()
	if err != nil {
		t.Fatalf("expected load ok with %d shards: %v", ExpectedRedisShardCount, err)
	}
}

func TestLoad_productionFilterTimeoutCeiling(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("SERVER_PORT", "8181")
	t.Setenv("DB_DSN", "postgres://u:p@localhost/db?sslmode=disable")
	t.Setenv("REDIS_ADDRS", "127.0.0.1:6479,127.0.0.1:6480,127.0.0.1:6481,127.0.0.1:6482")
	t.Setenv("TOKEN_SYMMETRIC_KEY", "01234567890123456789012345678901")
	t.Setenv("FILTER_TIMEOUT_MS", "101")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "FILTER_TIMEOUT_MS") {
		t.Fatalf("expected FILTER_TIMEOUT_MS error, got %v", err)
	}

	t.Setenv("FILTER_TIMEOUT_MS", "100")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.FilterTimeoutMs != 100 {
		t.Fatalf("FilterTimeoutMs=%d want 100", cfg.FilterTimeoutMs)
	}
}

func TestFraudScoringEnabled_defaultFalse(t *testing.T) {
	cfg := &Config{}
	if cfg.FraudScoringEnabled() {
		t.Fatal("FRAUD_SCORING_ENABLED must default to false")
	}

	t.Setenv("FRAUD_SCORING_ENABLED", "true")
	cfg2 := &Config{}
	cfg2.FraudScoring.Enabled = getEnvBool("FRAUD_SCORING_ENABLED", false)
	if !cfg2.FraudScoringEnabled() {
		t.Fatal("FRAUD_SCORING_ENABLED=true must enable scoring")
	}
}

func TestBrokerPrimaryCH(t *testing.T) {
	cfg := &Config{}
	cfg.Broker.CHIngestSource = "broker"
	if !cfg.BrokerPrimaryCH() {
		t.Fatal("broker CH ingest source must be primary")
	}
	cfg.Broker.CHIngestSource = ""
	if cfg.BrokerPrimaryCH() {
		t.Fatal("empty CH ingest source must not be broker-primary")
	}
}

func TestPostbackDefaults(t *testing.T) {
	t.Setenv("POSTBACK_POLL_INTERVAL_MS", "2000")
	t.Setenv("POSTBACK_BATCH_SIZE", "100")
	cfg := &Config{}
	loadPostbackModules(cfg)
	if cfg.Postback.PollIntervalMs != 2000 {
		t.Fatalf("poll interval: got %d want 2000", cfg.Postback.PollIntervalMs)
	}
	if cfg.Postback.BatchSize != 100 {
		t.Fatalf("batch size: got %d want 100", cfg.Postback.BatchSize)
	}
	if cfg.PostbackPollInterval() != 2*time.Second {
		t.Fatalf("poll duration: got %v", cfg.PostbackPollInterval())
	}
	if cfg.PostbackBatchSize() != 100 {
		t.Fatalf("batch int32: got %d", cfg.PostbackBatchSize())
	}
}

func TestBillingExportDefaults(t *testing.T) {
	t.Setenv("BILLING_EXPORT_FETCH_ROWS", "2000")
	t.Setenv("BILLING_EXPORT_JOB_TIMEOUT_MIN", "30")
	cfg := &Config{}
	loadControlplaneModules(cfg)
	if cfg.Billing.ExportFetchRows != 2000 {
		t.Fatalf("fetch rows: got %d", cfg.Billing.ExportFetchRows)
	}
	if cfg.Billing.ExportJobTimeoutMin != 30 {
		t.Fatalf("timeout min: got %d", cfg.Billing.ExportJobTimeoutMin)
	}
}

func TestValidateBrokerPrimaryRequiresBrokerURL(t *testing.T) {
	cfg := &Config{
		ServerPort:        "8181",
		DBDSN:             "postgres://localhost/db",
		TokenSymmetricKey: "test-key",
		RedisAddrs:        []string{"127.0.0.1:6379"},
	}
	cfg.Broker.CHIngestSource = "broker"
	if err := validateAndApplyDefaults(cfg); err == nil {
		t.Fatal("CH_INGEST_SOURCE=broker without BROKER_URL must fail validation")
	}
	cfg.Broker.URL = "/run/ad-event-processor/broker/gnet.sock"
	if err := validateAndApplyDefaults(cfg); err != nil {
		t.Fatalf("broker-primary with BROKER_URL must pass: %v", err)
	}
}

func TestResolveControlPort_prefersControlPort(t *testing.T) {
	t.Setenv("CONTROL_PORT", "9191")
	t.Setenv("MANAGEMENT_PORT", "8188")
	if got := resolveControlPort(); got != "9191" {
		t.Fatalf("resolveControlPort()=%q want 9191", got)
	}
	t.Setenv("CONTROL_PORT", "")
	if got := resolveControlPort(); got != "8188" {
		t.Fatalf("resolveControlPort()=%q want 8188 from MANAGEMENT_PORT", got)
	}
}

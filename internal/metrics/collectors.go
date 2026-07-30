package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_http_requests_total",
		Help: "Total number of HTTP requests by status code",
	}, []string{"method", "path", "status"})

	HttpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_http_request_duration_seconds",
		Help:    "Latency of HTTP requests in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	EventsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_events_processed_total",
		Help: "Total number of events successfully accepted into Redis Streams",
	})

	EventsDropped = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_events_dropped_total",
		Help: "Total number of events dropped due to Redis ingestion failure",
	})

	FilterBlockedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_filter_blocked_total",
		Help: "Total number of events blocked by filters",
	}, []string{"reason"})

	DbWriteDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_db_write_duration_seconds",
		Help:    "Duration of database batch write operations",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5},
	}, []string{"type"})

	DbWriteErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_db_write_errors_total",
		Help: "Total number of database write errors",
	}, []string{"type"})

	CircuitBreakerState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_circuit_breaker_state",
		Help: "Current state of the circuit breaker (0=closed, 1=open, 2=half-open)",
	}, []string{"group"})

	RedisBreakerState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_redis_breaker_state",
		Help: "Current state of the Redis shard circuit breaker (0=closed, 1=open, 2=half-open)",
	}, []string{"shard"})

	DlqSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_dlq_size_total",
		Help: "Current number of events in the Dead Letter Queue",
	})

	CommissionsCollectedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_management_commissions_total",
		Help: "Total amount of commissions collected from campaign cancellations",
	})

	BalanceTopupsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_management_topups_total",
		Help: "Total amount of customer balance top-ups",
	}, []string{"currency"})

	ActiveCampaigns = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_management_active_campaigns_count",
		Help: "Current number of active campaigns in the system",
	})

	DataDriftRatio = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_reconciliation_drift_ratio",
		Help: "Ratio of discrepancy between Postgres and ClickHouse spend",
	}, []string{"campaign_id"})

	ReconRunsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_reconciliation_runs_total",
		Help: "Total number of completed reconciliation runs",
	}, []string{"status"})

	ReconDiscrepanciesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_reconciliation_discrepancies_total",
		Help: "Total number of campaign discrepancies found",
	})

	ReconTotalDelta = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_reconciliation_total_delta_micro_units",
		Help: "Absolute net discrepancy corrected by reconciliation in micro units",
	})

	ReconAdjustmentErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_reconciliation_adjustment_errors_total",
		Help: "Total number of errors during automated reconciliation corrections",
	})
	ReconDriftMicro = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_reconciliation_drift_micro",
		Help: "Absolute micro-unit drift detected during budget snapshot reconciliation",
	}, []string{"campaign_id"})
	ReconCorrectionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_reconciliation_corrections_total",
		Help: "RECONCILIATION_ADJUST outbox events enqueued by recon workers",
	})
	ReconCorrectionsAppliedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_reconciliation_corrections_applied_total",
		Help: "RECONCILIATION_ADJUST outbox events applied successfully",
	})

	GnetPacketsReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_gnet_packets_received_total",
		Help: "Total number of network packets received",
	})
	GnetPacketsSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_gnet_packets_sent_total",
		Help: "Total number of network packets sent",
	})
	GnetActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_gnet_active_connections",
		Help: "Current number of active TCP connections",
	})
	GnetEventLoopWorkDuration = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_gnet_event_loop_work_duration_seconds_total",
		Help: "Total execution time spent doing active processing in gnet event loops",
	})
	GnetBytesReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_gnet_bytes_received_total",
		Help: "Total number of bytes received via gnet",
	})
	GnetBytesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_gnet_bytes_sent_total",
		Help: "Total number of bytes sent via gnet",
	})
	HttpParseErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_http_parse_errors_total",
		Help: "Total number of HTTP/1.1 parsing errors",
	}, []string{"error_type"})
	IngressLegacyJSONTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "espx_ingress_legacy_json_total",
		Help: "Track payloads that fell back to deprecated flat bid_micro / category_mask JSON",
	})
	WorkerPoolRejectTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_worker_pool_reject_total",
		Help: "Requests rejected because pinned worker pool queue is full",
	})

	HandlerLogDropTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_handler_log_drop_total",
		Help: "Accepted events whose audit log write was dropped (logger ring full)",
	})
	FraudStreamDropTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_fraud_stream_drop_total",
		Help: "Fraud reject events dropped because the async fraud stream ring is full",
	})
	FraudStreamCriticalDropTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_fraud_stream_critical_drop_total",
		Help: "Critical fraud lane (L1/L3) drops after short spin when critical ring is full",
	})
	FraudStreamAggregatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_fraud_stream_aggregated_total",
		Help: "Fraud reject events folded into subnet/reason aggregates during ring pressure",
	})
	FraudStreamAggregatedDropTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_fraud_stream_dropped_total",
		Help: "Fraud events dropped because the fixed aggregate hash table overflowed",
	})
	FraudStreamMode = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_fraud_stream_mode",
		Help: "Fraud stream writer mode (1 = aggregating at >=80% ring fill or force)",
	}, []string{"aggregating"})
	FraudStreamAggTableFill = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_fraud_stream_agg_table_fill_ratio",
		Help: "Ratio of occupied slots in the fraud aggregate hash table (0-1)",
	})
	FraudStreamRingFillRatio = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_fraud_stream_ring_fill_ratio",
		Help: "Analytical fraud ring fill ratio (0-1)",
	})
	FraudStreamPending = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_fraud_stream_pending",
		Help: "Pending fraud events across critical + analytical rings",
	})
	FraudStreamBackpressureTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_fraud_stream_backpressure_total",
		Help: "Times tracker forced fraud aggregation due to consumer lag signal",
	})
	FraudStreamPELAgeSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_fraud_stream_pel_age_seconds",
		Help: "Oldest pending entry idle age for ad:events:stream (or fraud stream) per shard",
	}, []string{"stream", "shard"})
	H2HostileDisconnectTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_h2_hostile_disconnect_total",
		Help: "HTTP/2 connections closed after H2_INCOMPLETE_MAX incomplete frames with consumed=0",
	})

	FilterThroughput = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_filter_throughput_total",
		Help: "Total throughput through the filter engine",
	}, []string{"format"})
	FilterDecisions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_filter_decisions_total",
		Help: "Filter decisions made by the engine",
	}, []string{"decision"})
	FilterInternalErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_filter_internal_errors_total",
		Help: "Non-fatal filter dependency failures (geo lookup, redis side-effects)",
	}, []string{"kind"})
	FilterGeoDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_filter_geo_duration_seconds",
		Help:    "Geo filter MaxMind country lookup duration (sampled 1/128 by default)",
		Buckets: []float64{0.000001, 0.000002, 0.000005, 0.00001, 0.000025, 0.00005, 0.0001, 0.00025, 0.0005, 0.001},
	})
	TTCBypassTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_ttc_bypass_total",
		Help: "Clicks accepted without impression timestamp (TTC fail-open bypass)",
	})
	FilterTierDegradedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "filter_tier_degraded_total",
		Help: "Filter checks that skipped non-critical Lua gates near the monotonic deadline",
	})

	RedisLuaDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_redis_lua_duration_seconds",
		Help:    "Execution duration of Redis Lua filters",
		Buckets: []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
	}, []string{"shard"})
	RedisLuaNoScriptTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_redis_lua_noscript_total",
		Help: "EVALSHA NOSCRIPT fallbacks to full EVAL (script evicted or not preloaded)",
	}, []string{"shard"})
	RedisLuaScriptLoaded = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_redis_lua_script_loaded",
		Help: "1 if filter_full (unified-filter) Lua is loaded on shard via SCRIPT LOAD, else 0",
	}, []string{"shard"})
	RedisLuaFastScriptLoaded = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_redis_lua_fast_script_loaded",
		Help: "1 if budget_fast Lua is loaded on shard via SCRIPT LOAD, else 0",
	}, []string{"shard"})
	RedisLuaFastPathTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_redis_lua_fast_path_total",
		Help: "Unified filter checks routed to budget_fast.lua",
	}, []string{"shard"})
	RedisLuaFullPathTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_redis_lua_full_path_total",
		Help: "Unified filter checks routed to filter_full (unified-filter) Lua",
	}, []string{"shard"})
	RedisLuaFastDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_redis_lua_fast_duration_seconds",
		Help:    "Execution duration of budget_fast Lua filters",
		Buckets: []float64{0.0001, 0.00025, 0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05},
	}, []string{"shard"})
	RedisOpsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_redis_ops_total",
		Help: "Unified-filter Redis EvalSha round trips per shard (includes budget-miss retries)",
	}, []string{"shard"})
	RedisCampaignOpsSampledTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_redis_campaign_ops_sampled_total",
		Help: "Sampled unified-filter Redis ops by campaign for per-shard top-N dashboards (see METRICS_HISTOGRAM_SAMPLE_MASK)",
	}, []string{"shard", "campaign_id"})
	TrackerCampaignSpendSampledTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_tracker_campaign_spend_micro_sampled_total",
		Help: "Sampled accepted spend in micro-units by campaign and shard for hot-campaign detection",
	}, []string{"shard", "campaign_id"})

	BudgetCacheWarmTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_budget_cache_warm_total",
		Help: "Redis budget:campaign:* keys inserted via SET NX during registry sync warm",
	}, []string{"type"})
	BudgetCacheMissTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_budget_cache_miss_total",
		Help: "Unified filter Lua budget key misses (return -1)",
	})
	BudgetCacheMissPGTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_budget_cache_miss_pg_total",
		Help: "Budget cache misses resolved via PostgreSQL GetByID on hot path",
	})
	BudgetCacheRegistryRecoverTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_budget_cache_registry_recover_total",
		Help: "Budget cache misses recovered from in-memory registry without PostgreSQL",
	})

	RegistrySyncLag = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_registry_sync_lag_seconds",
		Help:    "Registry sync lag between database update and cache loading",
		Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
	})
	RegistryWarmDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_registry_warm_duration_seconds",
		Help:    "Duration of incremental registry warm (UpdateAndWarmCampaign)",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 5.0},
	})
	RegistryStaleMode = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_registry_stale_mode",
		Help: "1 when tracker registry is in stale-serve mode (shard-0 pub/sub quiet > REGISTRY_STALE_TTL), else 0",
	})
	Shard0PubSubUnreachable = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_shard0_pubsub_unreachable",
		Help: "1 when shard-0 campaigns:update pub/sub is unreachable (tracker stale-serve), else 0",
	})

	GeoProviderStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_geo_provider_status",
		Help: "Status of the geo provider: 1 = real MaxMind, 0 = mock",
	})

	TrackerHealthDegraded = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_tracker_health_degraded",
		Help: "1 when tracker /health is DEGRADED (postgres or any redis shard ping failed), else 0",
	})
	TrackerRedisShardHealthy = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_tracker_redis_shard_healthy",
		Help: "Per-shard redis ping from tracker health probe: 1 healthy, 0 unreachable",
	}, []string{"shard"})

	ManagementOutboxPendingTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_management_outbox_pending_total",
		Help: "Count of outbox_events rows in PENDING status awaiting Redis propagation",
	})
	ManagementOutboxOldestPendingSeconds = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_management_outbox_oldest_pending_seconds",
		Help: "Age in seconds of the oldest PENDING outbox event (0 when queue empty)",
	})
	BlacklistReplicationLag = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_blacklist_replication_lag_seconds",
		Help:    "Outbox-to-Redis blacklist fan-out latency (HR-BL)",
		Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
	})

	ManagementOpsAlertEnqueueFailuresTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_management_ops_alert_enqueue_failures_total",
		Help: "Failed notifier enqueue attempts from OpsAlerter and Alertmanager webhook",
	})

	AdminFanoutSourcesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_admin_fanout_sources_total",
		Help: "Fan-out source polls per admin route",
	}, []string{"route"})
	AdminFanoutPartialTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_admin_fanout_partial_total",
		Help: "Fan-out responses with at least one source failure",
	}, []string{"route"})
	AdminFanoutLatencySeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_admin_fanout_latency_seconds",
		Help:    "End-to-end fan-out request latency per admin route",
		Buckets: prometheus.DefBuckets,
	}, []string{"route"})

	GeoIPUpdateErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_geoip_update_errors_total",
		Help: "MaxMind GeoIP database update failures",
	})

	GeoIPReloadErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_geoip_reload_errors_total",
		Help: "GeoIP hot-reload failures in the tracker watcher",
	})

	FraudScoreHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_fraud_score_histogram",
		Help:    "Accumulated fraud score (0-100) per scored request",
		Buckets: []float64{0, 15, 30, 45, 60, 75, 90, 100},
	})
	FraudTierTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_fraud_tier_total",
		Help: "Fraud score tier assignments after filter scoring",
	}, []string{"tier"})
	FraudReasonTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_fraud_reason_total",
		Help: "Fraud signal contributions by stable reason code",
	}, []string{"reason"})
	L1RejectTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_l1_reject_total",
		Help: "L1 auto-reject decisions (dual high-confidence or L3 blocklist)",
	})

	BrokerProduceTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_broker_produce_total",
		Help: "Broker produce attempts by topic and status",
	}, []string{"topic", "status"})
	BrokerFetchTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_broker_fetch_total",
		Help: "Broker fetch requests by topic",
	}, []string{"topic"})
	BrokerActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_broker_active_connections",
		Help: "Current number of active broker TCP client connections",
	})
	BrokerFsyncDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_broker_fsync_duration_seconds",
		Help:    "Duration of partition log fsync operations",
		Buckets: []float64{0.0001, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
	})
	BrokerReplicationLag = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_broker_replication_lag_messages",
		Help: "Leader log_hwm minus local next offset (messages behind on this node)",
	}, []string{"topic"})
	BrokerLeaderEpoch = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_broker_leader_epoch",
		Help: "Current fencing epoch when this node is leader for the topic (0 when follower)",
	}, []string{"topic", "node_id"})
	BrokerActiveLeader = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_broker_active_leader",
		Help: "1 when this node is elected leader for the topic, else 0",
	}, []string{"topic", "node_id"})
	BrokerLeaderReady = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_broker_leader_ready",
		Help: "1 when elected leader has caught up to log_hwm and may accept writes",
	}, []string{"topic", "node_id"})
	BrokerDiskWritable = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_broker_disk_writable",
		Help: "1 when the broker data directory is writable, else 0",
	})
	BrokerConnectionsRejected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_broker_connections_rejected_total",
		Help: "TCP connections closed because max-connections limit was reached",
	})
	BrokerReplicationErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_broker_replication_errors_total",
		Help: "Follower replication failures by topic and reason",
	}, []string{"topic", "reason"})
	BrokerProduceDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_broker_produce_duration_seconds",
		Help:    "End-to-end produce handler latency on the broker",
		Buckets: []float64{0.0001, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
	}, []string{"topic"})
	BrokerFetchDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_broker_fetch_duration_seconds",
		Help:    "End-to-end fetch handler latency on the broker",
		Buckets: []float64{0.0001, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
	}, []string{"topic"})
	BrokerLeaderElectionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_broker_leader_election_total",
		Help: "Leader term acquisitions per topic (SETNX wins with epoch bump)",
	}, []string{"topic"})
	BrokerReplicationCatchupSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_broker_replication_catchup_seconds",
		Help:    "Time for a new leader to become ready after failover",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30},
	}, []string{"topic"})
	BrokerRetentionDeletedSegments = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_broker_retention_deleted_segments_total",
		Help: "Sealed log segments removed by the retention worker",
	})
	BrokerRetentionDiskUsageBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_broker_retention_disk_usage_bytes",
		Help: "On-disk bytes for topic partition logs after the latest retention pass",
	}, []string{"topic"})
	BrokerRetentionOldestSegmentAgeSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_broker_retention_oldest_segment_age_seconds",
		Help: "Age of the oldest segment file for the topic",
	}, []string{"topic"})
	BrokerConsumerLagMessages = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_broker_consumer_lag_messages",
		Help: "Leader high watermark minus committed offset for a consumer group",
	}, []string{"topic", "group"})
	BrokerConsumerCommitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_broker_consumer_commits_total",
		Help: "Successful consumer offset commits",
	}, []string{"topic", "group"})
	BrokerIngestMessagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_broker_ingest_messages_total",
		Help: "Broker log records parsed by processor bridge",
	}, []string{"topic", "group", "event_type"})
	BrokerIngestParseErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_broker_ingest_parse_errors_total",
		Help: "Unrecognized broker payloads on the processor bridge",
	}, []string{"topic", "group"})
	BrokerIngestCommitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_broker_ingest_commits_total",
		Help: "Processor bridge offset commits after successful batch handling",
	}, []string{"topic", "group"})
	BrokerShadowMessagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_broker_shadow_messages_total",
		Help: "Broker events counted in shadow mode without store writes",
	}, []string{"topic", "group"})
	BrokerIngestDivergenceMessages = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_broker_ingest_divergence_messages",
		Help: "Redis stream length minus broker committed offset (shadow validation)",
	}, []string{"topic", "group"})
	BrokerIngestDivergenceHigh = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_broker_ingest_divergence_high",
		Help: "1 when broker/redis ingest divergence exceeds configured threshold",
	}, []string{"topic", "group"})

	DiskGateAppendWaitSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_disk_gate_append_wait_seconds",
		Help:    "Wait time acquiring the disk append semaphore by tier.",
		Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
	}, []string{"tier"})
	DiskGateFsyncInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_disk_gate_fsync_in_flight",
		Help: "Number of in-flight fsync operations (capacity 1).",
	})
	DiskGateShedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_disk_gate_shed_total",
		Help: "Total TierLow append attempts shed while degraded.",
	})
	DiskGateDegraded = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_disk_gate_degraded",
		Help: "Disk gate degraded state (1 = shedding TierLow).",
	})

	RegionProxyKeygenRate = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_region_proxy_keygen_rate",
		Help: "WAL records marked WalFlagDedupReady by the KeyGen thread.",
	})
	RegionProxyKeygenQueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_region_proxy_keygen_queue_depth",
		Help: "WAL records appended but not yet WalFlagDedupReady.",
	})
	RegionProxyKeygenLagSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_region_proxy_keygen_lag_seconds",
		Help:    "KeyGen factor_u derivation latency per record.",
		Buckets: []float64{0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
	})

	OpKeypoolDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_op_keypool_depth",
		Help: "OpKeyPool MPSC ring backlog.",
	})
	RegionProxyIngressShedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_region_proxy_ingress_shed_total",
		Help: "Ingress operations shed due to OpKeyPool depth above watermark.",
	})

	ControlScoreFallbackTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_control_score_fallback_total",
		Help: "Node capacity scorer fallbacks away from own_window by provenance.",
	}, []string{"provenance"})

	RtbAuctionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_rtb_auction_duration_seconds",
		Help:    "In-process RTB auction latency",
		Buckets: []float64{0.000001, 0.0000025, 0.000005, 0.00001, 0.000025, 0.00005, 0.0001, 0.00025, 0.0005, 0.001},
	})
	RtbAuctionNoBidTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_rtb_auction_no_bid_total",
		Help: "RTB auctions that did not clear a winner",
	}, []string{"reason"})
	RtbAuctionWinTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_rtb_auction_win_total",
		Help: "RTB auctions that cleared a winner and spent budget",
	})
	RtbAuctionCandidatesScanned = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_rtb_auction_candidates_scanned",
		Help:    "Campaign rows examined per auction in the geo shard",
		Buckets: []float64{1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500},
	})
	RtbAuctionScanLimitTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_rtb_auction_scan_limit_total",
		Help: "RTB auctions that hit the max candidate scan budget before clearing",
	})
	RtbBudgetSpendRejectedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_rtb_budget_spend_rejected_total",
		Help: "Final CAS budget debits rejected after a winner was selected",
	})
	RtbShadowWinnerMismatchTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_rtb_shadow_winner_mismatch_total",
		Help: "RTB shadow auctions where the eval winner differs from the client campaign_id",
	})
	RtbShadowNoBidTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_rtb_shadow_no_bid_total",
		Help: "RTB shadow eval returned no-bid while the client supplied a campaign_id",
	}, []string{"reason"})
	RtbShadowPriceDeltaMicro = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_rtb_shadow_price_delta_micro",
		Help:    "Absolute clearing price minus payload bid_micro on shadow wins (sampled)",
		Buckets: []float64{1, 10, 100, 1_000, 10_000, 100_000, 1_000_000, 10_000_000},
	})
	RtbBudgetReconcileDivergenceMicro = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_rtb_budget_reconcile_divergence_micro",
		Help:    "Absolute Redis minus RTB campaign budget delta on reconcile samples",
		Buckets: []float64{1, 10, 100, 1_000, 10_000, 100_000, 1_000_000, 10_000_000},
	})
	RtbBudgetReconcileHigh = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_rtb_budget_reconcile_high",
		Help: "1 when sampled Redis/RTB budget divergence exceeds configured threshold",
	})
	RtbBudgetReconcileSamplesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_rtb_budget_reconcile_samples_total",
		Help: "Campaign budget reconcile samples completed",
	})
	GlobalSpendBatchesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_global_spend_batches_total",
		Help: "Cross-region spend sync batches applied on the global cell",
	})
	GlobalSpendTxnsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_global_spend_txns_total",
		Help: "Cross-region spend sync transactions applied on the global cell",
	})
	GlobalSpendFlushErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_global_spend_flush_errors_total",
		Help: "Global spend reconciler flush failures",
	})
	GlobalSpendBatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_global_spend_batch_size",
		Help:    "Txn count per cross-region spend sync batch",
		Buckets: []float64{50, 100, 150, 200, 300, 500, 1000},
	})
	RegionSpendSyncBatchesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_region_spend_sync_batches_total",
		Help: "Spend sync batches appended to region-proxy WAL from regional processor",
	})
	RegionSpendSyncTxnsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_region_spend_sync_txns_total",
		Help: "Spend sync transactions staged for region-proxy uplink",
	})
	GlobalSpendApplyLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_global_spend_apply_latency_seconds",
		Help:    "Wall time to apply one cross-region spend sync batch (PG + Redis)",
		Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0},
	})
	TrackerLocalQuotaBlockTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_tracker_local_quota_block_total",
		Help: "Total number of events blocked locally by tracker quota cache",
	})
	LocalQuotaRefillTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_local_quota_refill_total",
		Help: "Local quanta refill attempts by outcome",
	}, []string{"status"})
	LocalQuotaRefillHerdTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_local_quota_refill_herd_total",
		Help: "Refill attempts rejected by per-shard concurrency cap (herd control)",
	})
	LocalQuotaShadowDiffTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_local_quota_shadow_diff_total",
		Help: "Shadow-mode local spend divergences from Lua debit",
	})
	LocalQuotaSpendTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_local_quota_spend_total",
		Help: "Successful local quanta debits on the hot path",
	})
	LocalQuotaFlushTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_local_quota_flush_total",
		Help: "Local quanta returned to Redis on pause/shutdown/strict",
	}, []string{"reason"})
	FilterLuaBranchTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "filter_lua_branch_total",
		Help: "Unified/budget-fast Lua return-code branch tags",
	}, []string{"branch"})
	FilterLuaSlowTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "filter_lua_slow_total",
		Help: "EVALSHA durations exceeding FILTER_SLOW_MS",
	})

	UDPControlPacketsReceivedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_udp_control_packets_received_total",
		Help: "UDP control datagrams received on tracker",
	})
	UDPControlPacketsAppliedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_udp_control_packets_applied_total",
		Help: "UDP control datagrams applied to ingress snapshot",
	})
	UDPControlCorruptTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_udp_control_corrupt_total",
		Help: "Malformed or invalid UDP control datagrams dropped",
	})
	UDPControlStaleDropTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_udp_control_stale_drop_total",
		Help: "Out-of-order UDP epochs dropped (epoch <= current)",
	})
	UDPControlStaleTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_udp_control_stale_total",
		Help: "UDP control channel marked STALE (no valid packet for 2x sync interval)",
	})
	UDPControlRecoveredTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_udp_control_recovered_total",
		Help: "UDP control channel recovered from STALE to OK",
	})
	UDPControlGapTightenTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_udp_control_gap_tighten_total",
		Help: "Epoch gap applied immediately because limits tightened",
	})
	UDPControlLoosenBlockedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_udp_control_loosen_blocked_total",
		Help: "Epoch gap loosen rejected without CONFIG_SNAPSHOT",
	})
	UDPControlSnapshotAppliedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_udp_control_snapshot_applied_total",
		Help: "CONFIG_SNAPSHOT epochs applied after gap or request",
	})
	UDPControlConfigRequestTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_udp_control_config_request_total",
		Help: "CONFIG_REQUEST datagrams sent by tracker",
	})
	UDPControlEpochLag = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_udp_control_epoch_lag",
		Help: "Management epoch minus tracker applied epoch",
	})
	UDPControlPublishTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_udp_control_publish_total",
		Help: "QUOTA_EPOCH / CONFIG_SNAPSHOT bursts sent by management",
	})
	RegionOutboxDeliveredTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_region_outbox_delivered_total",
		Help: "Outbox events applied to a regional Redis cell by RegionOutboxRelay",
	})
	DedupProposalTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_dedup_proposal_total",
		Help: "D3 v2 dedup_claim_confirm outcomes",
	}, []string{"status"})
	DedupMismatchTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_dedup_mismatch_total",
		Help: "D3 v2 hash_mismatch rejections (same SSID, different factor_u)",
	})
	DedupConfirmLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_dedup_confirm_latency_seconds",
		Help:    "dedup_claim_confirm round-trip latency",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 14),
	})
	RegionOutboxDeliveryLag = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_region_outbox_delivery_lag_seconds",
		Help:    "Outbox created_at to regional DELIVERED latency",
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 16),
	})
	OpLeaseBookedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_op_lease_booked_total",
		Help: "Operation leases booked per replica set",
	})
	OpLeaseExecutionTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_op_lease_execution_total",
		Help: "Operation lease CAS wins (booked to executing)",
	})
	OpLeaseExpiredTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_op_lease_expired_total",
		Help: "Operation leases transitioned to expired by the janitor",
	})
	OpBookedQueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_op_booked_queue_depth",
		Help: "Standby booked operation leases visible to this node",
	})
	OpLeaseHeartbeatRenewTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_op_lease_heartbeat_renew_total",
		Help: "Operation lease deadline extensions from executor heartbeat",
	})
	UDPIngressAcquireTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_udp_ingress_acquire_total",
		Help: "Ingress quota checks passed (lock-free per worker cell)",
	})
	UDPIngressRejectTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_udp_ingress_reject_total",
		Help: "Ingress quota rejections when per-shard worker cell exceeds epoch limit",
	})

	QuotaDriftDetectedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_quota_drift_detected_total",
		Help: "Campaign quota PG vs Redis drift events beyond chunk_size",
	})
	QuotaRepairEnqueuedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_quota_repair_enqueued_total",
		Help: "QUOTA_REPAIR outbox events enqueued by ReconWorker",
	})
	QuotaRepairAppliedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_quota_repair_applied_total",
		Help: "QUOTA_REPAIR outbox events applied successfully",
	})
	QuotaDeadShardReleaseTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_quota_dead_shard_release_total",
		Help: "PG quota rows released after dead-shard quorum confirmed",
	})

	ProcessorStreamLagSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_processor_stream_lag_seconds",
		Help: "Current stream processing lag in seconds per processor instance",
	}, []string{"instance"})
	ProcessorWeight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_processor_weight",
		Help: "Active consume weight for this processor instance (0-1)",
	}, []string{"instance"})
	MicroBatchPaused = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_micro_batch_paused",
		Help: "Whether the micro-batch scoring is paused due to stream lag (1=paused, 0=running)",
	})
	MicroBatchProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_micro_batch_processed_total",
		Help: "Total number of events processed by the micro-batcher",
	})
	MicroBatchBoostsWrittenTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_micro_batch_boosts_written_total",
		Help: "Total number of score boosts written to Redis from the micro-batcher",
	})
	CHSpoolAppendTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_ch_spool_append_total",
		Help: "ClickHouse batches durably spooled to mmap WAL during outages",
	})
	CHSpoolReplayTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_ch_spool_replay_total",
		Help: "ClickHouse spool WAL batches replayed after recovery",
	})
	CHSpoolRotateTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_ch_spool_rotate_total",
		Help: "ClickHouse spool segment rotations during long outages",
	})
	CHSpoolSegments = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_ch_spool_segments",
		Help: "Current ClickHouse spool segment count (active + sealed)",
	})
	ProcessorStreamXLen = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_processor_stream_xlen",
		Help: "Redis stream length (XLEN) per shard",
	}, []string{"shard"})
	ProcessorPgAcquireWaitSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_processor_pg_acquire_wait_seconds",
		Help:    "Wait time to acquire a processor-global Postgres write slot (alias of ad_processor_write_acquire_wait_seconds{backend=\"postgres\"})",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 16),
	})
	ProcessorWriteAcquireWaitSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_processor_write_acquire_wait_seconds",
		Help:    "Wait time to acquire a processor-global store write slot",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 16),
	}, []string{"backend"})
	ProcessorStreamBackpressureActive = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_processor_stream_backpressure_active",
		Help: "Stream consumer paused XREADGROUP while store circuit is open (1=active)",
	}, []string{"group"})

	EdgeBlocklistSkipAllowlistedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "edge_blocklist_skip_allowlisted_total",
		Help: "Total number of blocklist sync attempts skipped because the IP is allowlisted",
	})

	EdgeTarpitDelaySeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "edge_tarpit_delay_seconds",
		Help:    "Duration of tarpit delays introduced on suspicious requests",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 12),
	})

	VendorProbeSuccess = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_vendor_probe_success",
		Help: "Last vendor probe outcome (1=success, 0=failure)",
	}, []string{"vendor"})
	VendorProbeLatencySeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_vendor_probe_latency_seconds",
		Help:    "Vendor probe round-trip latency",
		Buckets: prometheus.ExponentialBuckets(0.05, 2, 12),
	}, []string{"vendor"})
	VendorProbeErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_vendor_probe_errors_total",
		Help: "Vendor probe failures (logged once per interval per vendor)",
	}, []string{"vendor"})

	CostSyncRunsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_cost_sync_runs_total",
		Help: "Cost sync runs by outcome (success, failed)",
	}, []string{"status"})
	CostSyncRowsImported = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_cost_sync_rows_imported_total",
		Help: "Campaign cost line items imported from network APIs",
	})
	CostSyncDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_cost_sync_duration_seconds",
		Help:    "Duration of one network cost sync run",
		Buckets: []float64{0.5, 1, 2, 5, 10, 30, 60, 90, 120},
	}, []string{"network"})
	CostSyncReconciliationDelta = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_cost_sync_reconciliation_delta_micro_total",
		Help: "Absolute micro-unit delta applied via RECONCILIATION_ADJUST entries",
	})
	CostSyncCHErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_cost_sync_ch_errors_total",
		Help: "ClickHouse cost_snapshots insert failures",
	})

	LedgerBatchPauseTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ledger_batch_pause_total",
		Help: "Campaigns paused after ledger batch flush partial failure or insufficient balance",
	})
	SyncLedgerBatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_sync_ledger_batch_size",
		Help:    "Campaign count per consolidated ledger flush Postgres transaction",
		Buckets: []float64{1, 2, 4, 8, 16, 32},
	})
	SyncLagSeconds = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_sync_lag_seconds",
		Help: "Seconds since campaign last PG spend update while inflight budget remains in Redis",
	})
	SyncLockExpiredTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sync_lock_expired_total",
		Help: "Budget sync lock TTL extensions or expirations detected during flush prepare",
	})
	OutboxPollIntervalMs = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_outbox_poll_interval_ms",
		Help:    "Outbox worker idle poll interval in milliseconds (coefficient backoff)",
		Buckets: []float64{20, 40, 80, 160, 250},
	})
	MgmtPgGateRejectedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_mgmt_pg_gate_rejected_total",
		Help: "Management Postgres gate rejections when LOW tier budget is exhausted",
	}, []string{"tier"})
	MgmtPgGateAcquireWaitSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ad_mgmt_pg_gate_acquire_wait_seconds",
		Help:    "Wait time acquiring a management Postgres gate slot",
		Buckets: prometheus.DefBuckets,
	}, []string{"tier"})

	CHDiskUsedPercent = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_ch_disk_used_percent",
		Help: "ClickHouse data volume used percent from system.disks",
	})
	CHJanitorRetentionDropTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_ch_janitor_retention_drop_total",
		Help: "Partitions dropped by CHPartitionJanitor retention policy",
	})
	CHJanitorEmergencyDropTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_ch_janitor_emergency_drop_total",
		Help: "Partitions dropped by CHPartitionJanitor emergency disk policy",
	})
	CHJanitorRecompressTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_ch_janitor_recompress_total",
		Help: "Partitions recompressed (OPTIMIZE FINAL) by CHPartitionJanitor off-peak pass",
	})

	CHQueryDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ad_ch_query_duration_seconds",
		Help:    "Duration of governed ClickHouse read queries",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 16),
	})
	CHQueryRejectedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_ch_query_rejected_total",
		Help: "Governed ClickHouse queries rejected because the concurrency gate was full",
	})
	CHActivePartsMax = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ad_ch_active_parts_max",
		Help: "Max active part count across table/partition (from system.parts); alert > 100",
	})
	CHSingleRowInsertsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_ch_single_row_inserts_total",
		Help: "ClickHouse store attempts narrowed to a single event during poison-pill binary split",
	})

	SlotMigrationLagMessages = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ad_slot_migration_lag_messages",
		Help: "Replication lag messages between dual-write source and target during slot migration",
	}, []string{"slot", "version"})
	SlotMigrationDualWriteTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_slot_migration_dual_write_total",
		Help: "Dual-write operations during zero-downtime slot migration cutover",
	}, []string{"slot", "result"})
	SlotMigrationCutoverBlockedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_slot_migration_cutover_blocked_total",
		Help: "Slot cutover attempts blocked (lag threshold, missing keys, invariant failure)",
	}, []string{"reason"})
	ElasticRoutingCutoverTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_elastic_routing_cutover_total",
		Help: "Global routing_epoch bumps with broker/TCP cutover",
	})
	ElasticCampaignMigrationTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_elastic_campaign_migration_total",
		Help: "Per-campaign elastic triplet migrations completed by ShardOrchestrator",
	})
	TCPControlSnapshotSentTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_tcp_control_snapshot_sent_total",
		Help: "Signed TCP routing snapshots sent to trackers",
	})
	TCPControlSnapshotAppliedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_tcp_control_snapshot_applied_total",
		Help: "Signed TCP routing snapshots applied on tracker",
	})
	TCPControlSnapshotErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_tcp_control_snapshot_errors_total",
		Help: "TCP routing snapshot encode/decode/HMAC failures",
	})
	TCPControlAckSentTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_tcp_control_ack_sent_total",
		Help: "Tracker ACK frames sent after TCP snapshot apply",
	})
	TCPControlAckReceivedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_tcp_control_ack_received_total",
		Help: "Management received tracker ACK after TCP snapshot",
	})

	XDPPassTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_xdp_pass_total",
		Help: "XDP packets passed to the kernel stack by reason",
	}, []string{"reason"})
	XDPDropTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ad_xdp_drop_total",
		Help: "XDP packets dropped at L4 edge by reason",
	}, []string{"reason"})
	XDPFingerprintTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_xdp_fingerprint_total",
		Help: "Passive SYN TCP fingerprints emitted to ringbuf (cold path, score only)",
	})
)

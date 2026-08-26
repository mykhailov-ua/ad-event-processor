package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func loadIngestModules(cfg *Config, appEnv string) error {
	cfg.Logger.Dir = os.Getenv("LOGGER_DIR")
	if cfg.Logger.Dir == "" {
		cfg.Logger.Dir = "/var/log/ad-event-processor"
	}
	cfg.Logger.Shards = getEnvInt("LOGGER_SHARDS", 8)
	cfg.Logger.FlushSizeKB = getEnvInt("LOGGER_FLUSH_SIZE_KB", 256)
	cfg.Logger.RotateSizeMB = getEnvInt("LOGGER_ROTATE_SIZE_MB", 512)
	cfg.Logger.RotateInterval = time.Duration(getEnvInt("LOGGER_ROTATE_INTERVAL_MIN", 60)) * time.Minute
	cfg.Logger.LatencyLimit = time.Duration(getEnvInt("LOGGER_LATENCY_LIMIT_MS", 100)) * time.Millisecond
	cfg.Logger.PersistQueueDepth = getEnvInt("LOGGER_PERSIST_QUEUE_DEPTH", 0)
	cfg.Logger.PersistEnqueueTimeout = time.Duration(getEnvInt("LOGGER_PERSIST_ENQUEUE_TIMEOUT_MS", 25)) * time.Millisecond

	cfg.Broker.URL = os.Getenv("BROKER_URL")
	cfg.Broker.RedisURL = os.Getenv("BROKER_REDIS_URL")
	cfg.Broker.Topic = os.Getenv("BROKER_TOPIC")
	cfg.Broker.FraudTopic = os.Getenv("BROKER_FRAUD_TOPIC")
	cfg.Broker.PartitionCount = getEnvInt("BROKER_PARTITION_COUNT", ExpectedRedisShardCount)
	cfg.Broker.ShadowMode = getEnvBool("BROKER_SHADOW_MODE", false)
	cfg.Broker.CHIngestSource = os.Getenv("CH_INGEST_SOURCE")
	cfg.Broker.MaxBytes = getEnvInt("BROKER_FETCH_MAX_BYTES", 1024*1024)
	cfg.Broker.TimeoutMs = getEnvInt("BROKER_TIMEOUT_MS", 5000)
	cfg.Broker.ReconcileIntervalMs = getEnvInt("BROKER_RECONCILE_INTERVAL_MS", 30000)
	cfg.Broker.DivergenceThreshold = uint64(getEnvInt64("BROKER_DIVERGENCE_THRESHOLD", 1000))
	cfg.Broker.ProducerCapacity = getEnvInt("BROKER_PRODUCER_CAPACITY", 32768)
	cfg.Broker.ProducerBatchSize = getEnvInt("BROKER_PRODUCER_BATCH_SIZE", 512)
	cfg.Broker.ProducerFlushMs = getEnvInt("BROKER_PRODUCER_FLUSH_MS", 2)
	if cfg.Broker.Topic == "" {
		cfg.Broker.Topic = "tracker-logs"
	}
	if cfg.Broker.FraudTopic == "" {
		cfg.Broker.FraudTopic = "ad-fraud-events"
	}

	cfg.RtbMode = os.Getenv("RTB_MODE")
	cfg.RtbBudgetAuthority = os.Getenv("RTB_BUDGET_AUTHORITY")
	cfg.RtbClearingMode = os.Getenv("RTB_CLEARING_MODE")
	cfg.RtbSnapshotPath = os.Getenv("RTB_SNAPSHOT_PATH")
	cfg.RtbHybridMaxRpsPerNode = getEnvInt("RTB_HYBRID_MAX_RPS_PER_NODE", 0)
	cfg.RtbReconcileIntervalMs = getEnvInt("RTB_RECONCILE_INTERVAL_MS", 30000)
	cfg.RtbBudgetDivergenceMicro = int64(getEnvInt("RTB_BUDGET_DIVERGENCE_THRESHOLD_MICRO", 1000))
	cfg.RtbReconcileSampleSize = getEnvInt("RTB_RECONCILE_SAMPLE_SIZE", 32)
	cfg.RtbTargetingIndex = getEnvBool("RTB_TARGETING_INDEX", true)
	cfg.RtbPrebidIVT = getEnvBool("RTB_PREBID_IVT", false)
	cfg.RtbExchangeMaxQPS = getEnvInt("RTB_EXCHANGE_MAX_QPS", 0)
	cfg.RtbExchangeMaxBodyBytes = getEnvInt64("RTB_EXCHANGE_MAX_BODY_BYTES", 1<<20)
	cfg.RtbExchangeNoBidMode = os.Getenv("RTB_EXCHANGE_NO_BID_MODE")
	if cfg.RtbExchangeNoBidMode == "" {
		cfg.RtbExchangeNoBidMode = "204"
	}
	cfg.RtbExchangeMultiImpMax = getEnvInt("RTB_EXCHANGE_MULTI_IMP_MAX", 1)
	cfg.RtbExchangeGzip = getEnvBool("RTB_EXCHANGE_GZIP", true)
	cfg.RtbExchangeDelivery = os.Getenv("RTB_EXCHANGE_DELIVERY")
	if cfg.RtbExchangeDelivery == "" {
		cfg.RtbExchangeDelivery = "adm"
	}
	cfg.RtbExchangeNURLTemplate = os.Getenv("RTB_EXCHANGE_NURL_TEMPLATE")
	cfg.TrackerTgClickBaseURL = os.Getenv("TRACKER_TG_CLICK_BASE_URL")
	if cfg.TrackerTgClickBaseURL == "" {
		cfg.TrackerTgClickBaseURL = "http://track.local/tg/click"
	}
	cfg.RtbExchangeSeatID = os.Getenv("RTB_EXCHANGE_SEAT_ID")
	if cfg.RtbExchangeSeatID == "" {
		cfg.RtbExchangeSeatID = "1"
	}
	cfg.RtbRegsPolicy = os.Getenv("RTB_REGS_POLICY")
	if cfg.RtbRegsPolicy == "" {
		cfg.RtbRegsPolicy = "flag"
	}
	cfg.RtbCoppaPolicy = os.Getenv("RTB_COPPA_POLICY")
	if cfg.RtbCoppaPolicy == "" {
		cfg.RtbCoppaPolicy = "flag"
	}
	cfg.RtbBlocklistEnforce = getEnvBool("RTB_BLOCKLIST_ENFORCE", true)
	cfg.RtbCatalogReloadSLOMs = getEnvInt("RTB_CATALOG_RELOAD_SLO_MS", 5000)
	cfg.RtbDealOutcomeFlushMs = getEnvInt("RTB_DEAL_OUTCOME_FLUSH_MS", 5000)
	if cfg.RtbBudgetAuthority == "" {
		cfg.RtbBudgetAuthority = "redis"
	}

	rawIngress := os.Getenv("TRACKER_INGRESS_SCHEMA")
	if rawIngress == "" {
		rawIngress = IngressSchemaOpenRTB3
	}
	cfg.IngressSchema = NormalizeIngressSchema(rawIngress)
	switch cfg.IngressSchema {
	case IngressSchemaOpenRTB3, IngressSchemaAdEventProcessorNative:
	default:
		return fmt.Errorf("invalid TRACKER_INGRESS_SCHEMA %q (want openrtb_3 or ad_event_processor_native)", cfg.IngressSchema)
	}

	cfg.QuotaMode = os.Getenv("QUOTA_MODE")
	if cfg.QuotaMode == "" {
		cfg.QuotaMode = "off"
	}
	cfg.LocalQuotaMode = os.Getenv("LOCAL_QUOTA_MODE")
	if cfg.LocalQuotaMode == "" {
		cfg.LocalQuotaMode = "off"
	}
	cfg.QuotaChunkSize = getEnvInt64("QUOTA_CHUNK_SIZE", 0)
	cfg.QuotaStrictThresholdMicro = getEnvInt64("QUOTA_STRICT_THRESHOLD_MICRO", 5_000_000)
	cfg.QuotaStrictExitMicro = getEnvInt64("QUOTA_STRICT_EXIT_MICRO", 8_000_000)
	cfg.QuotaRefillThresholdPct = getEnvInt("QUOTA_REFILL_THRESHOLD_PCT", 20)
	cfg.LocalQuotaRefillMaxShard = getEnvInt("LOCAL_QUOTA_REFILL_MAX_PER_SHARD", 4)
	cfg.QuotaAdaptiveFloorMicro = getEnvInt64("QUOTA_ADAPTIVE_FLOOR_MICRO", 500_000)
	cfg.QuotaAdaptiveCeilingMicro = getEnvInt64("QUOTA_ADAPTIVE_CEILING_MICRO", 50_000_000)
	cfg.BudgetDeltaTopic = os.Getenv("BUDGET_DELTA_TOPIC")
	if cfg.BudgetDeltaTopic == "" {
		cfg.BudgetDeltaTopic = "budget-deltas"
	}

	cfg.CIDRBlockEnabled = getEnvBool("CIDR_BLOCK_ENABLED", true)
	cfg.CIDRFeedDir = os.Getenv("CIDR_FEED_DIR")
	if cfg.CIDRFeedDir == "" {
		cfg.CIDRFeedDir = "/var/lib/ad-event-processor/cidr"
	}
	cfg.CIDRFeedRefresh = 24 * time.Hour
	if raw := os.Getenv("CIDR_FEED_REFRESH_INTERVAL"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			cfg.CIDRFeedRefresh = d
		} else if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			cfg.CIDRFeedRefresh = time.Duration(n) * time.Second
		}
	}
	cfg.CIDRFeedURLAWS = os.Getenv("CIDR_FEED_URL_AWS")
	cfg.CIDRFeedURLGCP = os.Getenv("CIDR_FEED_URL_GCP")
	cfg.CIDRFeedURLAzure = os.Getenv("CIDR_FEED_URL_AZURE")
	cfg.CIDRFeedURLTor = os.Getenv("CIDR_FEED_URL_TOR")
	cfg.CIDRFeedDownloadEnable = getEnvBool("CIDR_FEED_DOWNLOAD_ENABLED", false)

	cfg.IPv6RotationEnabled = getEnvBool("IPV6_ROTATION_ENABLED", true)
	cfg.IPv6RotationMode = os.Getenv("IPV6_ROTATION_MODE")
	if cfg.IPv6RotationMode == "" {
		cfg.IPv6RotationMode = "live"
	}
	cfg.IPv6RotationWindow = time.Minute
	if raw := os.Getenv("IPV6_ROTATION_WINDOW"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			cfg.IPv6RotationWindow = d
		}
	}
	cfg.IPv6RotationThreshold = uint32(getEnvInt("IPV6_ROTATION_THRESHOLD", 6))

	cfg.IPv4RotationEnabled = getEnvBool("IPV4_ROTATION_ENABLED", true)
	cfg.IPv4RotationMode = os.Getenv("IPV4_ROTATION_MODE")
	if cfg.IPv4RotationMode == "" {
		cfg.IPv4RotationMode = "live"
	}
	cfg.IPv4RotationWindow = time.Minute
	if raw := os.Getenv("IPV4_ROTATION_WINDOW"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			cfg.IPv4RotationWindow = d
		}
	}
	cfg.IPv4RotationThreshold = uint32(getEnvInt("IPV4_ROTATION_THRESHOLD", 6))

	cfg.CGNATMobileIPBypass = getEnvBool("CGNAT_MOBILE_IP_BYPASS", false)
	cfg.CGNATMobileCarrierASNs = os.Getenv("CGNAT_MOBILE_CARRIER_ASNS")

	cfg.OSFingerprintMismatchEnabled = getEnvBool("OS_FINGERPRINT_MISMATCH_ENABLED", true)

	cfg.DCASNHotEnabled = getEnvBool("DC_ASN_HOT_ENABLED", true)
	cfg.DCASNFeedDir = os.Getenv("DC_ASN_FEED_DIR")
	if cfg.DCASNFeedDir == "" {
		cfg.DCASNFeedDir = "/var/lib/ad-event-processor/dc-asn"
	}
	cfg.DCASNFeedRefresh = 24 * time.Hour
	if raw := os.Getenv("DC_ASN_FEED_REFRESH"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			cfg.DCASNFeedRefresh = d
		}
	}
	cfg.DCASNSampleMask = getEnvInt("DC_ASN_SAMPLE_MASK", -1)

	cfg.ResidentialProxyHotEnabled = getEnvBool("RESIDENTIAL_PROXY_HOT_ENABLED", true)
	cfg.ResidentialProxyWindow = 5 * time.Minute
	if raw := os.Getenv("RESIDENTIAL_PROXY_WINDOW"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			cfg.ResidentialProxyWindow = d
		}
	}

	cfg.TCPMSSAnomalyEnabled = getEnvBool("TCP_MSS_ANOMALY_ENABLED", true)
	cfg.TCPMSSAnomalyMinByte = uint8(getEnvInt("TCP_MSS_ANOMALY_MIN_BYTE", 2))

	cfg.ProxyVPNBlockEnabled = getEnvBool("PROXY_VPN_BLOCK_ENABLED", true)
	cfg.ProxyVPNFeedDir = os.Getenv("PROXY_VPN_FEED_DIR")
	if cfg.ProxyVPNFeedDir == "" {
		cfg.ProxyVPNFeedDir = "/var/lib/ad-event-processor/proxy-vpn"
	}
	cfg.ProxyVPNFeedRefresh = 24 * time.Hour
	cfg.TLSFingerprintEnabled = getEnvBool("TLS_FINGERPRINT_ENABLED", true)
	cfg.TLSFingerprintFeedDir = os.Getenv("TLS_FINGERPRINT_FEED_DIR")
	if cfg.TLSFingerprintFeedDir == "" {
		cfg.TLSFingerprintFeedDir = "/var/lib/ad-event-processor/tls-fingerprint"
	}
	cfg.TLSFingerprintFeedRefresh = 24 * time.Hour
	if raw := os.Getenv("TLS_FINGERPRINT_FEED_REFRESH_INTERVAL"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil {
			cfg.TLSFingerprintFeedRefresh = d
		} else if n, err := strconv.Atoi(raw); err == nil {
			cfg.TLSFingerprintFeedRefresh = time.Duration(n) * time.Second
		}
	}
	if raw := os.Getenv("LINK_SIGNING_HMAC_SECRET"); raw != "" {
		cfg.LinkSigningHMACSecret = Secret(raw)
	}
	if raw := os.Getenv("ATTESTATION_HMAC_SECRET"); raw != "" {
		cfg.AttestationHMACSecret = Secret(raw)
	}
	if raw := os.Getenv("ATTESTATION_HMAC_SECRET_PREV"); raw != "" {
		cfg.AttestationHMACSecretPrev = Secret(raw)
	}
	if raw := os.Getenv("PROXY_VPN_FEED_REFRESH_INTERVAL"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			cfg.ProxyVPNFeedRefresh = d
		} else if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			cfg.ProxyVPNFeedRefresh = time.Duration(n) * time.Second
		}
	}

	cfg.ModeratorIntelEnabled = getEnvBool("MODERATOR_INTEL_ENABLED", false)
	cfg.ModeratorIntelFeedDir = strings.TrimSpace(os.Getenv("MODERATOR_INTEL_FEED_DIR"))
	if cfg.ModeratorIntelFeedDir == "" {
		cfg.ModeratorIntelFeedDir = "/var/lib/ad-event-processor/moderator-intel"
	}
	cfg.ModeratorIntelFeedRefresh = 24 * time.Hour
	if raw := strings.TrimSpace(os.Getenv("MODERATOR_INTEL_FEED_REFRESH_INTERVAL")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			cfg.ModeratorIntelFeedRefresh = d
		}
	}
	cfg.ModeratorIntelFeedURL = strings.TrimSpace(os.Getenv("MODERATOR_INTEL_FEED_URL"))
	cfg.ModeratorIntelFeedSecret = strings.TrimSpace(os.Getenv("MODERATOR_INTEL_FEED_SECRET"))
	cfg.ModeratorIntelFeedDownload = getEnvBool("MODERATOR_INTEL_FEED_DOWNLOAD", false)
	cfg.ModeratorIntelAllowUnsigned = getEnvBool("MODERATOR_INTEL_ALLOW_UNSIGNED", false)

	cfg.DomainPoolEnabled = getEnvBool("DOMAIN_POOL_ENABLED", true)
	cfg.DomainPoolSyncInterval = 30 * time.Second
	if raw := strings.TrimSpace(os.Getenv("DOMAIN_POOL_SYNC_INTERVAL")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			cfg.DomainPoolSyncInterval = d
		}
	}
	cfg.FlowRoutingEnabled = getEnvBool("FLOW_ROUTING_ENABLED", true)
	cfg.FlowSyncInterval = 30 * time.Second
	if raw := strings.TrimSpace(os.Getenv("FLOW_SYNC_INTERVAL")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			cfg.FlowSyncInterval = d
		}
	}

	cfg.ProxyAllowHTTPInsecure = getEnvBool("PROXY_ALLOW_HTTP_INSECURE", false)

	cfg.SlotMapReloadTopic = os.Getenv("SLOT_MAP_RELOAD_TOPIC")
	if cfg.SlotMapReloadTopic == "" {
		cfg.SlotMapReloadTopic = "shards:reload"
	}
	cfg.SlotMapPollIntervalMs = getEnvInt("SLOT_MAP_POLL_INTERVAL_MS", 10000)
	cfg.SlotMigrationEnabled = getEnvBool("SLOT_MIGRATION_ENABLED", true)
	cfg.SlotMigrationIntervalMs = getEnvInt("SLOT_MIGRATION_INTERVAL_MS", 30000)
	cfg.MigrationFenceEnabled = getEnvBool("MIGRATION_FENCE_ENABLED", appEnv == "production")
	cfg.SlotMigrationDualWriteEnabled = getEnvBool("SLOT_MIGRATION_DUAL_WRITE_ENABLED", false)
	cfg.SlotMigrationLagEpsilon = getEnvInt64("SLOT_MIGRATION_LAG_EPSILON", 0)
	cfg.SlotMigrationLagThreshold = getEnvInt64("SLOT_MIGRATION_LAG_THRESHOLD", 1000)
	return nil
}

func ModeratorIntelEnvExplicitlySet() bool {
	_, ok := os.LookupEnv("MODERATOR_INTEL_ENABLED")
	return ok
}

func ApplyModeratorIntelWhenEntitled(cfg *Config, entitled bool) {
	if cfg == nil || !entitled || ModeratorIntelEnvExplicitlySet() {
		return
	}
	cfg.ModeratorIntelEnabled = true
}

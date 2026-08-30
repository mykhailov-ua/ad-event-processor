package stream

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/metrics"
)

//go:embed processor_ch_ingest_policy.json
var processorClickHouseIngestPolicyEmbed []byte

const sealedProcessorClickHouseIngestAssetLabel = licensing.AssetLabelProcessorClickHouseIngest

var processorCHIngestPolicyActive atomic.Pointer[ProcessorClickHouseIngestPolicy]

type ProcessorClickHouseIngestPolicy struct {
	Version         int  `json:"version"`
	WALSegmentMBMax int  `json:"wal_segment_mb_max"`
	Compress        bool `json:"compress"`
}

// InitProcessorClickHouseIngestPolicy loads embedded processor_ch_ingest_policy.json or
// license-sealed blob (AD_EVENT_PROCESSOR_PROCESSOR_CH_INGEST_SEALED_BLOB). Caps WAL
// segment size via ApplyClickHouseIngestPolicy on spool startup.
func InitProcessorClickHouseIngestPolicy() error {
	raw, err := resolveProcessorClickHouseIngestPolicyBytes()
	if err != nil {
		return err
	}
	var policy ProcessorClickHouseIngestPolicy
	if err := json.Unmarshal(raw, &policy); err != nil {
		return fmt.Errorf("processor ch ingest policy json: %w", err)
	}
	if policy.Version < 1 {
		return fmt.Errorf("processor ch ingest policy version: %d", policy.Version)
	}
	processorCHIngestPolicyActive.Store(&policy)
	return nil
}

func ProcessorClickHouseIngestPolicyLoaded() (ProcessorClickHouseIngestPolicy, bool) {
	p := processorCHIngestPolicyActive.Load()
	if p == nil {
		return ProcessorClickHouseIngestPolicy{}, false
	}
	return *p, true
}

func ApplyClickHouseIngestPolicy(cfg ClickHouseSpoolConfig) ClickHouseSpoolConfig {
	policy, ok := ProcessorClickHouseIngestPolicyLoaded()
	if !ok || policy.WALSegmentMBMax <= 0 {
		return cfg
	}
	maxBytes := int64(policy.WALSegmentMBMax) * 1024 * 1024
	if cfg.SegmentSizeBytes > maxBytes {
		cfg.SegmentSizeBytes = maxBytes
	}
	return cfg
}

func sealedProcessorClickHouseIngestBlobPath() string {
	if v := os.Getenv("AD_EVENT_PROCESSOR_PROCESSOR_CH_INGEST_SEALED_BLOB"); v != "" {
		return v
	}
	return filepath.Join("internal", "ingestion", "processor_ch_ingest_sealed.bin")
}

func sealedProcessorClickHouseIngestBlob() ([]byte, error) {
	path := sealedProcessorClickHouseIngestBlobPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func resolveProcessorClickHouseIngestPolicyBytes() ([]byte, error) {
	if config.LicenseAssetsUnsealed() {
		return processorClickHouseIngestPolicyEmbed, nil
	}
	sealed, err := sealedProcessorClickHouseIngestBlob()
	if err != nil {
		if os.IsNotExist(err) {
			return processorClickHouseIngestPolicyEmbed, nil
		}
		return nil, err
	}
	mck, err := licensing.DeriveMCKFromLicenseFile(
		config.LicensePathFromEnv(),
		nil,
		licensing.HostFingerprint(),
	)
	if err != nil {
		metrics.ProcessorClickHouseIngestSealFailTotal.Inc()
		return nil, fmt.Errorf("sealed processor ch ingest mck: %w", err)
	}
	plain, err := licensing.OpenAsset(sealedProcessorClickHouseIngestAssetLabel, sealed, mck)
	if err != nil {
		metrics.ProcessorClickHouseIngestSealFailTotal.Inc()
		return nil, fmt.Errorf("sealed processor ch ingest open: %w", err)
	}
	if len(plain) == 0 {
		metrics.ProcessorClickHouseIngestSealFailTotal.Inc()
		return nil, licensing.ErrSealFormat
	}
	return plain, nil
}

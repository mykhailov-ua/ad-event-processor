package ingestion

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
var processorCHIngestPolicyEmbed []byte

const sealedProcessorCHIngestAssetLabel = licensing.AssetLabelProcessorCHIngest

var processorCHIngestPolicyActive atomic.Pointer[ProcessorCHIngestPolicy]

// ProcessorCHIngestPolicy is the cold-path ClickHouse ingest policy opened from a sealed blob.
type ProcessorCHIngestPolicy struct {
	Version         int  `json:"version"`
	WALSegmentMBMax int  `json:"wal_segment_mb_max"`
	Compress        bool `json:"compress"`
}

// InitProcessorCHIngestPolicy loads the processor CH ingest policy (sealed or dev embed).
func InitProcessorCHIngestPolicy() error {
	raw, err := resolveProcessorCHIngestPolicyBytes()
	if err != nil {
		return err
	}
	var policy ProcessorCHIngestPolicy
	if err := json.Unmarshal(raw, &policy); err != nil {
		return fmt.Errorf("processor ch ingest policy json: %w", err)
	}
	if policy.Version < 1 {
		return fmt.Errorf("processor ch ingest policy version: %d", policy.Version)
	}
	processorCHIngestPolicyActive.Store(&policy)
	return nil
}

// ProcessorCHIngestPolicyLoaded returns the active policy when InitProcessorCHIngestPolicy succeeded.
func ProcessorCHIngestPolicyLoaded() (ProcessorCHIngestPolicy, bool) {
	p := processorCHIngestPolicyActive.Load()
	if p == nil {
		return ProcessorCHIngestPolicy{}, false
	}
	return *p, true
}

// ApplyCHIngestPolicy clamps spool config using the sealed processor policy when loaded.
func ApplyCHIngestPolicy(cfg CHSpoolConfig) CHSpoolConfig {
	policy, ok := ProcessorCHIngestPolicyLoaded()
	if !ok || policy.WALSegmentMBMax <= 0 {
		return cfg
	}
	maxBytes := int64(policy.WALSegmentMBMax) * 1024 * 1024
	if cfg.SegmentSizeBytes > maxBytes {
		cfg.SegmentSizeBytes = maxBytes
	}
	return cfg
}

func sealedProcessorCHIngestBlobPath() string {
	if v := os.Getenv("AD_EVENT_PROCESSOR_PROCESSOR_CH_INGEST_SEALED_BLOB"); v != "" {
		return v
	}
	return filepath.Join("internal", "ingestion", "processor_ch_ingest_sealed.bin")
}

func sealedProcessorCHIngestBlob() ([]byte, error) {
	path := sealedProcessorCHIngestBlobPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func resolveProcessorCHIngestPolicyBytes() ([]byte, error) {
	if config.LicenseAssetsUnsealed() {
		return processorCHIngestPolicyEmbed, nil
	}
	sealed, err := sealedProcessorCHIngestBlob()
	if err != nil {
		if os.IsNotExist(err) {
			return processorCHIngestPolicyEmbed, nil
		}
		return nil, err
	}
	mck, err := licensing.DeriveMCKFromLicenseFile(
		config.LicensePathFromEnv(),
		nil,
		licensing.HostFingerprint(),
	)
	if err != nil {
		metrics.ProcessorCHIngestSealFailTotal.Inc()
		return nil, fmt.Errorf("sealed processor ch ingest mck: %w", err)
	}
	plain, err := licensing.OpenAsset(sealedProcessorCHIngestAssetLabel, sealed, mck)
	if err != nil {
		metrics.ProcessorCHIngestSealFailTotal.Inc()
		return nil, fmt.Errorf("sealed processor ch ingest open: %w", err)
	}
	if len(plain) == 0 {
		metrics.ProcessorCHIngestSealFailTotal.Inc()
		return nil, licensing.ErrSealFormat
	}
	return plain, nil
}

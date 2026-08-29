package regionproxy

import (
	"context"
	"fmt"

	"ad-event-processor/internal/dedup"
	"ad-event-processor/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BatchInput struct {
	RegionCode  uint8
	NodeID      string
	SourceEpoch uint32
	Seq         uint64
	FactorU     uuid.UUID
	Payload     []byte
	OpID        uuid.UUID
}

type BatchResult struct {
	Outcome  dedup.Outcome
	DedupKey string
}

type BatchHost interface {
	Pool() *pgxpool.Pool
	MultiRegionGlobal() bool
	ApplyRegionSpendSyncBatch(ctx context.Context, batchDedupKey string, payload []byte) error
	EnsureProxyBatchBookAndExecute(ctx context.Context, in BatchInput, apply func(ctx context.Context, claim dedup.ClaimResult) error) (BatchResult, error)
}

func IngestBatch(ctx context.Context, host BatchHost, in BatchInput) (BatchResult, error) {
	if in.RegionCode == 0 {
		return BatchResult{}, fmt.Errorf("region ingest batch region=%d: invalid region code", in.RegionCode)
	}
	if in.NodeID == "" {
		return BatchResult{}, fmt.Errorf("region ingest batch region=%d: node_id required", in.RegionCode)
	}
	if len(in.Payload) == 0 {
		return BatchResult{}, fmt.Errorf("region ingest batch region=%d: empty payload", in.RegionCode)
	}

	var canonBuf [4096 + 64]byte
	canon := dedupkey.WriteCanonicalProxyBatchPayload(canonBuf[:0], in.Seq, in.Payload)
	expected := dedupkey.FactorU(canon)
	if expected != in.FactorU {
		return BatchResult{}, fmt.Errorf("region ingest batch region=%d: factor_u mismatch", in.RegionCode)
	}

	if host != nil && host.MultiRegionGlobal() {
		return ingestBatchLeased(ctx, host, in)
	}
	return ingestBatchDirect(ctx, host, in)
}

func ingestBatchDirect(ctx context.Context, host BatchHost, in BatchInput) (BatchResult, error) {
	pool := host.Pool()
	epoch := in.SourceEpoch
	if epoch == 0 && pool != nil {
		epoch = dedup.LoadRoutingEpoch(ctx, pool)
	}
	adapter := dedup.NewAdapter(pool, in.RegionCode, epoch)
	if adapter == nil {
		return BatchResult{}, fmt.Errorf("region ingest batch region=%d: dedup adapter unavailable", in.RegionCode)
	}

	seq := int64(in.Seq)
	scope := adapter.RegionScope(dedupkey.ProxySourceID(in.RegionCode, in.NodeID), seq, seq)
	claim, err := adapter.ClaimConfirm(ctx, scope, in.FactorU)
	if err != nil {
		return BatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
	}
	if guardErr := dedup.GuardOutcome(claim); guardErr != nil {
		return BatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, guardErr)
	}
	if claim.ShouldApply() {
		if err := adapter.RecordApply(ctx, claim.DedupKey); err != nil {
			return BatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
		}
		if err := host.ApplyRegionSpendSyncBatch(ctx, claim.DedupKey, in.Payload); err != nil {
			return BatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
		}
	}
	return BatchResult{Outcome: claim.Outcome, DedupKey: claim.DedupKey}, nil
}

func ingestBatchLeased(ctx context.Context, host BatchHost, in BatchInput) (BatchResult, error) {
	result, err := host.EnsureProxyBatchBookAndExecute(ctx, in, func(ctx context.Context, claim dedup.ClaimResult) error {
		if claim.ShouldApply() {
			return host.ApplyRegionSpendSyncBatch(ctx, claim.DedupKey, in.Payload)
		}
		return nil
	})
	if err != nil {
		return BatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
	}
	return result, nil
}

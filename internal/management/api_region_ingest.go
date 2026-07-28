package management

import (
	"context"
	"fmt"

	"espx/internal/dedup"
	db "espx/internal/ingestion/sqlc"
	"espx/pkg/dedupkey"

	"github.com/google/uuid"
)

// RegionIngestBatchInput is one region-proxy WAL record forwarded to global ingest.
type RegionIngestBatchInput struct {
	RegionCode  uint8
	NodeID      string
	SourceEpoch uint32
	Seq         uint64
	FactorU     uuid.UUID
	Payload     []byte
	OpID        uuid.UUID
}

// RegionIngestBatchResult is the D3 claim outcome for one ingest batch.
type RegionIngestBatchResult struct {
	Outcome  dedup.Outcome
	DedupKey string
}

// IngestRegionProxyBatch claims and records one proxy batch on the global cell.
func (s *Service) IngestRegionProxyBatch(ctx context.Context, in RegionIngestBatchInput) (RegionIngestBatchResult, error) {
	if in.RegionCode == 0 {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: invalid region code", in.RegionCode)
	}
	if in.NodeID == "" {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: node_id required", in.RegionCode)
	}
	if len(in.Payload) == 0 {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: empty payload", in.RegionCode)
	}

	var canonBuf [4096 + 64]byte
	canon := dedupkey.WriteCanonicalProxyBatchPayload(canonBuf[:0], in.Seq, in.Payload)
	expected := dedupkey.FactorU(canon)
	if expected != in.FactorU {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: factor_u mismatch", in.RegionCode)
	}

	if s != nil && s.cfg != nil && s.cfg.MultiRegionGlobal() {
		return s.ingestRegionProxyBatchLeased(ctx, in)
	}
	return s.ingestRegionProxyBatchDirect(ctx, in)
}

func (s *Service) ingestRegionProxyBatchDirect(ctx context.Context, in RegionIngestBatchInput) (RegionIngestBatchResult, error) {
	epoch := in.SourceEpoch
	if epoch == 0 && s.pool != nil {
		epoch = dedup.LoadRoutingEpoch(ctx, s.pool)
	}
	adapter := dedup.NewAdapter(s.pool, in.RegionCode, epoch)
	if adapter == nil {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: dedup adapter unavailable", in.RegionCode)
	}

	seq := int64(in.Seq)
	scope := adapter.RegionScope(dedupkey.ProxySourceID(in.RegionCode, in.NodeID), seq, seq)
	claim, err := adapter.ClaimConfirm(ctx, scope, in.FactorU)
	if err != nil {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
	}
	if guardErr := dedup.GuardOutcome(claim); guardErr != nil {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, guardErr)
	}
	if claim.ShouldApply() {
		if err := adapter.RecordApply(ctx, claim.DedupKey); err != nil {
			return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
		}
		if err := s.applyRegionSpendSyncBatch(ctx, claim.DedupKey, in.Payload); err != nil {
			return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
		}
	}
	return RegionIngestBatchResult{
		Outcome:  claim.Outcome,
		DedupKey: claim.DedupKey,
	}, nil
}

func (s *Service) ingestRegionProxyBatchLeased(ctx context.Context, in RegionIngestBatchInput) (RegionIngestBatchResult, error) {
	worker := s.OperationLeaseWorker()
	if worker == nil {
		worker = NewOperationLeaseWorker(s)
	}
	bookReq := ProxyBatchBookRequest(ctx, s, in, 1)
	if _, err := worker.EnsureBook(ctx, bookReq); err != nil {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
	}

	var result RegionIngestBatchResult
	err := worker.ExecuteOp(ctx, bookReq.OpID, func(ctx context.Context, _ db.OperationLease, claim dedup.ClaimResult) error {
		result = RegionIngestBatchResult{
			Outcome:  claim.Outcome,
			DedupKey: claim.DedupKey,
		}
		if claim.ShouldApply() {
			return s.applyRegionSpendSyncBatch(ctx, claim.DedupKey, in.Payload)
		}
		return nil
	})
	if err != nil {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
	}
	return result, nil
}

package controlplane

import (
	"context"
	"errors"
	"fmt"

	"espx/internal/dedup"
	"espx/internal/domain"
	db "espx/internal/domain/db"
	"espx/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) leaseReplicaNodes() []string {
	if s != nil && s.cfg != nil && s.cfg.NodeID != "" {
		return []string{s.cfg.NodeID}
	}
	return []string{"management"}
}

func (s *Service) OperationLeaseWorker() *OperationLeaseWorker {
	if s == nil {
		return nil
	}
	return s.leaseWorker
}

func (w *OperationLeaseWorker) EnsureBook(ctx context.Context, req OperationLeaseBookRequest) (OperationLeaseBookResult, error) {
	empty := OperationLeaseBookResult{}
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return empty, fmt.Errorf("operation lease ensure book op_id=%s: worker unavailable", req.OpID)
	}
	q := db.New(w.svc.pool)
	existing, err := q.GetOperationLease(ctx, domain.ToUUID(req.OpID))
	if err == nil {
		status, qerr := w.quorumStatus(ctx, req.OpID)
		if qerr != nil {
			return empty, qerr
		}
		status.Lease = existing
		return status, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return empty, fmt.Errorf("operation lease ensure book op_id=%s: %w", req.OpID, err)
	}
	result, err := w.Book(ctx, req)
	if errors.Is(err, ErrLeaseQuorumNotMet) {
		return result, nil
	}
	return result, err
}

func RelayDeliveryBookRequest(s *Service, regionCode uint8, outboxEventID int64, eventType string, payload []byte, attempt int32) OperationLeaseBookRequest {
	adapter := s.dedupAdapter()
	scope := dedupkey.Scope{}
	if adapter != nil {
		scope = adapter.RegionScope(dedupkey.RelaySourceID(regionCode), outboxEventID, outboxEventID)
	}
	factorU := dedupkey.FactorU(dedupkey.CanonicalRelayPayload(outboxEventID, eventType, payload))
	if attempt <= 0 {
		attempt = 1
	}
	return OperationLeaseBookRequest{
		OpID:         RelayDeliveryOpID(regionCode, outboxEventID),
		RegionCode:   int16(regionCode),
		Role:         "management",
		ReplicaSetID: uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("espx-relay-set:%d", regionCode))),
		Attempt:      attempt,
		FactorU:      factorU,
		Scope:        scope,
		ReplicaNodes: s.leaseReplicaNodes(),
	}
}

func ProxyBatchBookRequest(ctx context.Context, s *Service, in RegionIngestBatchInput, attempt int32) OperationLeaseBookRequest {
	epoch := in.SourceEpoch
	if epoch == 0 && s.pool != nil {
		epoch = dedup.LoadRoutingEpoch(ctx, s.pool)
	}
	adapter := dedup.NewAdapter(s.pool, in.RegionCode, epoch)
	seq := int64(in.Seq)
	scope := adapter.RegionScope(dedupkey.ProxySourceID(in.RegionCode, in.NodeID), seq, seq)
	if attempt <= 0 {
		attempt = 1
	}
	return OperationLeaseBookRequest{
		OpID:         ProxyBatchOpID(in.RegionCode, in.NodeID, in.Seq, in.OpID),
		RegionCode:   int16(in.RegionCode),
		Role:         "region-proxy",
		ReplicaSetID: uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("espx-proxy-set:%d:%s", in.RegionCode, in.NodeID))),
		Attempt:      attempt,
		FactorU:      in.FactorU,
		Scope:        scope,
		ReplicaNodes: []string{in.NodeID},
		BookAckNodes: []string{in.NodeID},
	}
}

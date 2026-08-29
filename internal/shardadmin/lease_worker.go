package shardadmin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"ad-event-processor/internal/dedup"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	defaultOpLeaseJanitorPeriod = 5 * time.Second
	defaultOpLeasePollInterval  = 200 * time.Millisecond
	defaultOpLeaseExpireBatch   = int32(500)
)

type OperationLeaseBookRequest struct {
	OpID         uuid.UUID
	RegionCode   int16
	Role         string
	ReplicaSetID uuid.UUID
	Attempt      int32
	FactorU      uuid.UUID
	Scope        dedupkey.Scope
	ReplicaNodes []string
	BookAckNodes []string
}

type OperationLeaseBookResult struct {
	Lease     db.OperationLease
	AckCount  int32
	Quorum    int32
	QuorumMet bool
}

type OperationLeaseExecuteFunc func(ctx context.Context, lease db.OperationLease, claim dedup.ClaimResult) error

type OperationLeaseWorker struct {
	host           LeaseHost
	nodeID         string
	role           string
	region         int16
	timeoutSec     int
	maxRenewals    int32
	janitorPeriod  time.Duration
	pollInterval   time.Duration
	fencing        *LeaseFencingRegistry
	opKeyGate      OpKeyPoolGate
	executor       OperationLeaseExecuteFunc
	renewHeartbeat bool
	onRenew        LeaseRenewHook
}

func NewOperationLeaseWorker(host LeaseHost) *OperationLeaseWorker {
	nodeID, _ := os.Hostname()
	timeoutSec := 30
	maxRenewals := int32(3)
	role := "management"
	region := int16(0)
	fencingDir := ""
	if host != nil {
		cfg := host.LeaseWorkerConfig()
		if cfg.NodeID != "" {
			nodeID = cfg.NodeID
		}
		if cfg.NodeRole != "" {
			role = cfg.NodeRole
		}
		region = cfg.RegionCode
		if cfg.OpLeaseTimeoutSec > 0 {
			timeoutSec = cfg.OpLeaseTimeoutSec
		}
		if cfg.OpLeaseMaxRenewals > 0 {
			maxRenewals = cfg.OpLeaseMaxRenewals
		}
		fencingDir = cfg.OpLeaseFencingDir
	}
	if fencingDir == "" {
		fencingDir = filepath.Join(os.TempDir(), "ad-event-processor-op-lease", nodeID)
	}
	fencing, err := NewLeaseFencingRegistry(fencingDir)
	if err != nil {
		slog.Warn("operation lease fencing registry init failed", "dir", fencingDir, "err", err)
	}
	return &OperationLeaseWorker{
		host:           host,
		nodeID:         nodeID,
		role:           role,
		region:         region,
		timeoutSec:     timeoutSec,
		maxRenewals:    maxRenewals,
		janitorPeriod:  defaultOpLeaseJanitorPeriod,
		pollInterval:   defaultOpLeasePollInterval,
		fencing:        fencing,
		renewHeartbeat: true,
	}
}

func (w *OperationLeaseWorker) SetExecutor(fn OperationLeaseExecuteFunc) {
	if w == nil {
		return
	}
	w.executor = fn
}

func (w *OperationLeaseWorker) SetOpKeyPoolGate(gate OpKeyPoolGate) {
	if w == nil {
		return
	}
	w.opKeyGate = gate
}

func (w *OperationLeaseWorker) Book(ctx context.Context, req OperationLeaseBookRequest) (OperationLeaseBookResult, error) {
	empty := OperationLeaseBookResult{}
	if w == nil || w.host == nil || w.host.Pool() == nil {
		return empty, fmt.Errorf("operation lease book op_id=%s: worker unavailable", req.OpID)
	}
	if w.opKeyGate != nil && w.opKeyGate.ShouldShed() {
		return empty, ErrOpKeyPoolShed
	}
	if req.OpID == uuid.Nil {
		return empty, fmt.Errorf("operation lease book: op_id required")
	}
	if req.ReplicaSetID == uuid.Nil {
		return empty, fmt.Errorf("operation lease book op_id=%s: replica_set_id required", req.OpID)
	}
	if req.Attempt <= 0 {
		req.Attempt = 1
	}
	if len(req.ReplicaNodes) == 0 {
		req.ReplicaNodes = []string{w.nodeID}
	}
	ackNodes := req.BookAckNodes
	if len(ackNodes) == 0 && len(req.ReplicaNodes) == 1 {
		ackNodes = req.ReplicaNodes
	}

	scopeRaw, err := EncodeLeaseDedupScope(req.Scope, req.Attempt)
	if err != nil {
		return empty, fmt.Errorf("operation lease book op_id=%s: %w", req.OpID, err)
	}

	if !w.postgresAvailable(ctx) {
		return w.bookRedis(ctx, req)
	}

	var lease db.OperationLease
	err = w.host.WithPostgresHigh(ctx, func(runCtx context.Context) error {
		q := db.New(w.host.Pool())
		booked, err := q.BookOperationLease(runCtx, db.BookOperationLeaseParams{
			OpID:         domain.ToUUID(req.OpID),
			RegionCode:   req.RegionCode,
			Role:         req.Role,
			ReplicaSetID: domain.ToUUID(req.ReplicaSetID),
			Attempt:      req.Attempt,
			FactorU:      domain.ToUUID(req.FactorU),
			DedupScope:   scopeRaw,
			TimeoutSec:   int32(w.timeoutSec),
		})
		if err != nil {
			return err
		}
		for _, nodeID := range req.ReplicaNodes {
			if err := q.InsertOperationLeaseReplica(runCtx, db.InsertOperationLeaseReplicaParams{
				OpID:   domain.ToUUID(req.OpID),
				NodeID: nodeID,
			}); err != nil {
				return err
			}
		}
		for _, nodeID := range ackNodes {
			if err := q.UpsertOperationLeaseReplicaBookAck(runCtx, db.UpsertOperationLeaseReplicaBookAckParams{
				OpID:   domain.ToUUID(req.OpID),
				NodeID: nodeID,
			}); err != nil {
				return err
			}
		}
		lease = booked
		return nil
	})
	if err != nil {
		if !w.postgresAvailable(ctx) || isPostgresUnavailable(err) {
			return w.bookRedis(ctx, req)
		}
		return empty, fmt.Errorf("operation lease book op_id=%s: %w", req.OpID, err)
	}

	result, qerr := w.quorumStatus(ctx, req.OpID)
	if qerr != nil {
		return empty, fmt.Errorf("operation lease book op_id=%s: %w", req.OpID, qerr)
	}
	result.Lease = lease
	if result.QuorumMet {
		metrics.OpLeaseBookedTotal.Inc()
		return result, nil
	}
	return result, ErrLeaseQuorumNotMet
}

func (w *OperationLeaseWorker) AckBook(ctx context.Context, opID uuid.UUID, nodeID string) (OperationLeaseBookResult, error) {
	empty := OperationLeaseBookResult{}
	if w == nil || w.host == nil || w.host.Pool() == nil {
		return empty, fmt.Errorf("operation lease ack book op_id=%s: worker unavailable", opID)
	}
	if nodeID == "" {
		nodeID = w.nodeID
	}
	if !w.postgresAvailable(ctx) {
		return w.ackBookRedis(ctx, opID, nodeID, 3)
	}
	err := w.host.WithPostgresHigh(ctx, func(runCtx context.Context) error {
		return db.New(w.host.Pool()).UpsertOperationLeaseReplicaBookAck(runCtx, db.UpsertOperationLeaseReplicaBookAckParams{
			OpID:   domain.ToUUID(opID),
			NodeID: nodeID,
		})
	})
	if err != nil {
		if !w.postgresAvailable(ctx) || isPostgresUnavailable(err) {
			return w.ackBookRedis(ctx, opID, nodeID, 3)
		}
		return empty, fmt.Errorf("operation lease ack book op_id=%s: %w", opID, err)
	}
	result, err := w.quorumStatus(ctx, opID)
	if err != nil {
		return empty, err
	}
	if result.QuorumMet {
		metrics.OpLeaseBookedTotal.Inc()
	}
	return result, nil
}

func (w *OperationLeaseWorker) quorumStatus(ctx context.Context, opID uuid.UUID) (OperationLeaseBookResult, error) {
	if !w.postgresAvailable(ctx) {
		return w.quorumStatusRedis(ctx, opID, 3)
	}
	q := db.New(w.host.Pool())
	opUUID := domain.ToUUID(opID)
	replicaCount, err := q.CountOperationLeaseReplicas(ctx, opUUID)
	if err != nil {
		if isPostgresUnavailable(err) {
			return w.quorumStatusRedis(ctx, opID, 3)
		}
		return OperationLeaseBookResult{}, err
	}
	ackCount, err := q.CountOperationLeaseBookAcks(ctx, opUUID)
	if err != nil {
		if isPostgresUnavailable(err) {
			return w.quorumStatusRedis(ctx, opID, int(replicaCount))
		}
		return OperationLeaseBookResult{}, err
	}
	quorum := QuorumRequired(int(replicaCount))
	return OperationLeaseBookResult{
		AckCount:  ackCount,
		Quorum:    quorum,
		QuorumMet: ackCount >= quorum,
	}, nil
}

func (w *OperationLeaseWorker) ExecuteOp(ctx context.Context, opID uuid.UUID, execute OperationLeaseExecuteFunc) error {
	if w == nil || w.host == nil || w.host.Pool() == nil {
		return fmt.Errorf("operation lease execute op_id=%s: worker unavailable", opID)
	}
	if execute == nil {
		return fmt.Errorf("operation lease execute op_id=%s: executor required", opID)
	}

	quorum, err := w.quorumStatus(ctx, opID)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	if !quorum.QuorumMet {
		return nil
	}

	opUUID := domain.ToUUID(opID)
	preLease, err := db.New(w.host.Pool()).GetOperationLease(ctx, opUUID)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	replicaSetID := uuid.UUID(preLease.ReplicaSetID.Bytes)

	fencingEpoch, err := w.nextFencingEpoch(replicaSetID)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}

	var won bool
	err = w.host.WithPostgresHigh(ctx, func(runCtx context.Context) error {
		q := db.New(w.host.Pool())
		rows, err := q.OperationLeaseClaimExecuting(runCtx, db.OperationLeaseClaimExecutingParams{
			OpID:         opUUID,
			NodeID:       w.nodeID,
			FencingEpoch: fencingEpoch,
		})
		if err != nil {
			return err
		}
		won = len(rows) > 0
		return nil
	})
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	if !won {
		return nil
	}
	metrics.OpLeaseExecutionTotal.Inc()

	lease, err := db.New(w.host.Pool()).GetOperationLease(ctx, opUUID)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	if err := AuthoritativeLeaseView(lease, w.nodeID, time.Now().UTC()); err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	if err := w.fencing.Validate(replicaSetID, lease.FencingEpoch); err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}

	scope, err := DedupScopeForLease(lease)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	factorU := uuid.UUID(lease.FactorU.Bytes)

	adapter := w.host.DedupAdapter(ctx)
	if adapter == nil {
		return fmt.Errorf("operation lease execute op_id=%s: dedup adapter unavailable", opID)
	}

	claim, err := adapter.ClaimConfirm(ctx, scope, factorU)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	if guardErr := dedup.GuardOutcome(claim); guardErr != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, guardErr)
	}

	applySideEffects := claim.ShouldApply()
	if claim.Outcome == dedup.OutcomeAlreadyConfirmed {
		resume, resumeErr := adapter.NeedsResumeApply(ctx, claim.DedupKey)
		if resumeErr != nil {
			return fmt.Errorf("operation lease execute op_id=%s: %w", opID, resumeErr)
		}
		applySideEffects = resume
	}
	if applySideEffects {
		lease, err = db.New(w.host.Pool()).GetOperationLease(ctx, opUUID)
		if err != nil {
			return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
		}
		if err := AuthoritativeLeaseView(lease, w.nodeID, time.Now().UTC()); err != nil {
			return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
		}
		if err := w.fencing.Validate(replicaSetID, lease.FencingEpoch); err != nil {
			return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
		}
		done := make(chan struct{})
		if w.renewHeartbeat {
			go w.runRenewHeartbeat(ctx, opID, done)
		}
		err = execute(ctx, lease, claim)
		close(done)
		if err != nil {
			return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
		}
		if err := adapter.RecordApply(ctx, claim.DedupKey); err != nil {
			return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
		}
	}

	_, err = db.New(w.host.Pool()).CompleteOperationLease(ctx, opUUID)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	return nil
}

func (w *OperationLeaseWorker) RenewLease(ctx context.Context, opID uuid.UUID) (db.OperationLease, error) {
	if w == nil || w.host == nil || w.host.Pool() == nil {
		return db.OperationLease{}, fmt.Errorf("operation lease renew op_id=%s: worker unavailable", opID)
	}
	var renewed db.OperationLease
	err := w.host.WithPostgresHigh(ctx, func(runCtx context.Context) error {
		row, err := db.New(w.host.Pool()).RenewOperationLease(runCtx, db.RenewOperationLeaseParams{
			OpID:           domain.ToUUID(opID),
			TimeoutSec:     int32(w.timeoutSec),
			ExecutorNodeID: pgtype.Text{String: w.nodeID, Valid: true},
			MaxRenewals:    w.maxRenewals,
		})
		if err != nil {
			return err
		}
		renewed = row
		return nil
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.OperationLease{}, ErrLeaseRenewExhausted
		}
		return db.OperationLease{}, fmt.Errorf("operation lease renew op_id=%s: %w", opID, err)
	}
	metrics.OpLeaseHeartbeatRenewTotal.Inc()
	if w.onRenew != nil {
		w.onRenew(opID)
	}
	return renewed, nil
}

func (w *OperationLeaseWorker) nextFencingEpoch(replicaSetID uuid.UUID) (int64, error) {
	if w.fencing == nil {
		return 1, nil
	}
	return w.fencing.Next(replicaSetID)
}

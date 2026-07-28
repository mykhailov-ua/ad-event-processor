package management

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"espx/internal/dedup"
	"espx/internal/ingestion"
	db "espx/internal/ingestion/sqlc"
	"espx/internal/metrics"
	"espx/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	defaultOpLeaseJanitorPeriod = 5 * time.Second
	defaultOpLeasePollInterval  = 200 * time.Millisecond
	defaultOpLeaseExpireBatch   = int32(500)
)

// OperationLeaseBookRequest books one replicated operation lease row.
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

// OperationLeaseBookResult is the outcome of a book attempt including quorum status (C3).
type OperationLeaseBookResult struct {
	Lease     db.OperationLease
	AckCount  int32
	Quorum    int32
	QuorumMet bool
}

// OperationLeaseExecuteFunc runs irreversible side effects after ClaimConfirm.
type OperationLeaseExecuteFunc func(ctx context.Context, lease db.OperationLease, claim dedup.ClaimResult) error

// OperationLeaseWorker books, claims, and completes replicated cold-path operations.
type OperationLeaseWorker struct {
	svc            *Service
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

// NewOperationLeaseWorker constructs a lease worker for the local management cell.
func NewOperationLeaseWorker(svc *Service) *OperationLeaseWorker {
	nodeID, _ := os.Hostname()
	timeoutSec := 30
	maxRenewals := int32(3)
	role := "management"
	region := int16(0)
	fencingDir := ""
	if svc != nil && svc.cfg != nil {
		if svc.cfg.NodeID != "" {
			nodeID = svc.cfg.NodeID
		}
		if svc.cfg.NodeRole != "" {
			role = svc.cfg.NodeRole
		}
		region = int16(svc.cfg.RegionCode)
		if svc.cfg.OpLeaseTimeoutSec > 0 {
			timeoutSec = svc.cfg.OpLeaseTimeoutSec
		}
		if svc.cfg.OpLeaseMaxRenewals > 0 {
			maxRenewals = int32(svc.cfg.OpLeaseMaxRenewals)
		}
		fencingDir = svc.cfg.OpLeaseFencingDir
	}
	if fencingDir == "" {
		fencingDir = filepath.Join(os.TempDir(), "espx-op-lease", nodeID)
	}
	fencing, err := NewLeaseFencingRegistry(fencingDir)
	if err != nil {
		slog.Warn("operation lease fencing registry init failed", "dir", fencingDir, "error", err)
	}
	return &OperationLeaseWorker{
		svc:            svc,
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

// SetExecutor registers the side-effect handler for polled booked leases.
func (w *OperationLeaseWorker) SetExecutor(fn OperationLeaseExecuteFunc) {
	if w == nil {
		return
	}
	w.executor = fn
}

// SetOpKeyPoolGate wires OpKeyPool backpressure for book shedding (C7).
func (w *OperationLeaseWorker) SetOpKeyPoolGate(gate OpKeyPoolGate) {
	if w == nil {
		return
	}
	w.opKeyGate = gate
}

// Book inserts a lease with PG-authoritative deadline_at (C4) and replica rows.
func (w *OperationLeaseWorker) Book(ctx context.Context, req OperationLeaseBookRequest) (OperationLeaseBookResult, error) {
	empty := OperationLeaseBookResult{}
	if w == nil || w.svc == nil || w.svc.pool == nil {
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

	if !w.pgAvailable(ctx) {
		return w.bookRedis(ctx, req)
	}

	var lease db.OperationLease
	err = w.svc.withPgHigh(ctx, func(runCtx context.Context) error {
		q := db.New(w.svc.pool)
		booked, err := q.BookOperationLease(runCtx, db.BookOperationLeaseParams{
			OpID:         ingestion.ToUUID(req.OpID),
			RegionCode:   req.RegionCode,
			Role:         req.Role,
			ReplicaSetID: ingestion.ToUUID(req.ReplicaSetID),
			Attempt:      req.Attempt,
			FactorU:      ingestion.ToUUID(req.FactorU),
			DedupScope:   scopeRaw,
			TimeoutSec:   int32(w.timeoutSec),
		})
		if err != nil {
			return err
		}
		for _, nodeID := range req.ReplicaNodes {
			if err := q.InsertOperationLeaseReplica(runCtx, db.InsertOperationLeaseReplicaParams{
				OpID:   ingestion.ToUUID(req.OpID),
				NodeID: nodeID,
			}); err != nil {
				return err
			}
		}
		for _, nodeID := range ackNodes {
			if err := q.UpsertOperationLeaseReplicaBookAck(runCtx, db.UpsertOperationLeaseReplicaBookAckParams{
				OpID:   ingestion.ToUUID(req.OpID),
				NodeID: nodeID,
			}); err != nil {
				return err
			}
		}
		lease = booked
		return nil
	})
	if err != nil {
		if !w.pgAvailable(ctx) || isPgUnavailable(err) {
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

// AckBook records a replica book ACK and returns quorum status (C3).
func (w *OperationLeaseWorker) AckBook(ctx context.Context, opID uuid.UUID, nodeID string) (OperationLeaseBookResult, error) {
	empty := OperationLeaseBookResult{}
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return empty, fmt.Errorf("operation lease ack book op_id=%s: worker unavailable", opID)
	}
	if nodeID == "" {
		nodeID = w.nodeID
	}
	if !w.pgAvailable(ctx) {
		return w.ackBookRedis(ctx, opID, nodeID, 3)
	}
	err := w.svc.withPgHigh(ctx, func(runCtx context.Context) error {
		return db.New(w.svc.pool).UpsertOperationLeaseReplicaBookAck(runCtx, db.UpsertOperationLeaseReplicaBookAckParams{
			OpID:   ingestion.ToUUID(opID),
			NodeID: nodeID,
		})
	})
	if err != nil {
		if !w.pgAvailable(ctx) || isPgUnavailable(err) {
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
	if !w.pgAvailable(ctx) {
		return w.quorumStatusRedis(ctx, opID, 3)
	}
	q := db.New(w.svc.pool)
	opUUID := ingestion.ToUUID(opID)
	replicaCount, err := q.CountOperationLeaseReplicas(ctx, opUUID)
	if err != nil {
		if isPgUnavailable(err) {
			return w.quorumStatusRedis(ctx, opID, 3)
		}
		return OperationLeaseBookResult{}, err
	}
	ackCount, err := q.CountOperationLeaseBookAcks(ctx, opUUID)
	if err != nil {
		if isPgUnavailable(err) {
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

// ExecuteOp claims executing, runs ClaimConfirm, applies side effects, RecordApply, and completes.
func (w *OperationLeaseWorker) ExecuteOp(ctx context.Context, opID uuid.UUID, execute OperationLeaseExecuteFunc) error {
	if w == nil || w.svc == nil || w.svc.pool == nil {
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

	opUUID := ingestion.ToUUID(opID)
	preLease, err := db.New(w.svc.pool).GetOperationLease(ctx, opUUID)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	replicaSetID := uuid.UUID(preLease.ReplicaSetID.Bytes)

	fencingEpoch, err := w.nextFencingEpoch(replicaSetID)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}

	var won bool
	err = w.svc.withPgHigh(ctx, func(runCtx context.Context) error {
		q := db.New(w.svc.pool)
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

	lease, err := db.New(w.svc.pool).GetOperationLease(ctx, opUUID)
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

	adapter := w.svc.dedupAdapter()
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
		lease, err = db.New(w.svc.pool).GetOperationLease(ctx, opUUID)
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

	_, err = db.New(w.svc.pool).CompleteOperationLease(ctx, opUUID)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	return nil
}

// RenewLease extends deadline_at for the executing holder (C6).
func (w *OperationLeaseWorker) RenewLease(ctx context.Context, opID uuid.UUID) (db.OperationLease, error) {
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return db.OperationLease{}, fmt.Errorf("operation lease renew op_id=%s: worker unavailable", opID)
	}
	var renewed db.OperationLease
	err := w.svc.withPgHigh(ctx, func(runCtx context.Context) error {
		row, err := db.New(w.svc.pool).RenewOperationLease(runCtx, db.RenewOperationLeaseParams{
			OpID:           ingestion.ToUUID(opID),
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

// RunJanitor expires stale booked/executing leases (C8 leader election).
func (w *OperationLeaseWorker) RunJanitor(ctx context.Context) (int32, error) {
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return 0, nil
	}
	ok, err := w.tryJanitorLock(ctx)
	if err != nil {
		return 0, fmt.Errorf("operation lease janitor: %w", err)
	}
	if !ok {
		return 0, nil
	}
	defer w.releaseJanitorLock(ctx)

	var expired int32
	err = w.svc.withPgLow(ctx, func(runCtx context.Context) error {
		n, err := db.New(w.svc.pool).OperationLeaseExpireStale(runCtx, defaultOpLeaseExpireBatch)
		if err != nil {
			return err
		}
		expired = n
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("operation lease janitor: %w", err)
	}
	if expired > 0 {
		metrics.OpLeaseExpiredTotal.Add(float64(expired))
	}
	return expired, nil
}

func (w *OperationLeaseWorker) tryJanitorLock(ctx context.Context) (bool, error) {
	var ok bool
	err := w.svc.pool.QueryRow(ctx,
		`SELECT pg_try_advisory_lock($1::int, $2::int)`,
		opLeaseJanitorLockKey1, int32(w.region),
	).Scan(&ok)
	return ok, err
}

func (w *OperationLeaseWorker) releaseJanitorLock(ctx context.Context) {
	_, _ = w.svc.pool.Exec(ctx,
		`SELECT pg_advisory_unlock($1::int, $2::int)`,
		opLeaseJanitorLockKey1, int32(w.region),
	)
}

// ProcessBooked drains booked leases assigned to this node through the registered executor.
func (w *OperationLeaseWorker) ProcessBooked(ctx context.Context) error {
	if w == nil || w.svc == nil || w.executor == nil {
		return nil
	}
	if w.opKeyGate != nil && w.opKeyGate.ShouldShed() {
		return nil
	}
	q := db.New(w.svc.pool)
	rows, err := q.ListBookedOperationLeasesForNode(ctx, db.ListBookedOperationLeasesForNodeParams{
		NodeID:   w.nodeID,
		RowLimit: 32,
	})
	if err != nil {
		return fmt.Errorf("operation lease process booked node=%s: %w", w.nodeID, err)
	}
	metrics.OpBookedQueueDepth.Set(float64(len(rows)))
	for _, row := range rows {
		opID := uuid.UUID(row.OpID.Bytes)
		if err := w.ExecuteOp(ctx, opID, w.executor); err != nil {
			if errors.Is(err, ErrStaleFencingEpoch) {
				slog.Warn("operation lease stale fencing", "op_id", opID)
				continue
			}
			slog.Warn("operation lease execute failed", "op_id", opID, "error", err)
		}
	}
	return nil
}

// Start runs the poll + janitor loops until ctx is cancelled.
func (w *OperationLeaseWorker) Start(ctx context.Context) {
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return
	}
	slog.Info("operation lease worker starting",
		"node_id", w.nodeID,
		"role", w.role,
		"region", w.region,
		"timeout_sec", w.timeoutSec,
		"janitor_period", w.janitorPeriod,
	)
	pollTicker := time.NewTicker(w.pollInterval)
	defer pollTicker.Stop()
	janitorTicker := time.NewTicker(w.janitorPeriod)
	defer janitorTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			if err := w.ProcessBooked(ctx); err != nil && ctx.Err() == nil {
				slog.Error("operation lease poll failed", "node_id", w.nodeID, "error", err)
			}
		case <-janitorTicker.C:
			if _, err := w.RunJanitor(ctx); err != nil && ctx.Err() == nil {
				slog.Error("operation lease janitor failed", "node_id", w.nodeID, "error", err)
			}
		}
	}
}

// LeaseOpID extracts the UUID from a sqlc lease row.
func LeaseOpID(lease db.OperationLease) uuid.UUID {
	if lease.OpID.Valid {
		return uuid.UUID(lease.OpID.Bytes)
	}
	return uuid.Nil
}

// LeaseFactorU extracts factor_u from a sqlc lease row.
func LeaseFactorU(lease db.OperationLease) uuid.UUID {
	if lease.FactorU.Valid {
		return uuid.UUID(lease.FactorU.Bytes)
	}
	return uuid.Nil
}

// LeaseDeadline returns the PG deadline when present.
func LeaseDeadline(lease db.OperationLease) time.Time {
	if lease.DeadlineAt.Valid {
		return lease.DeadlineAt.Time
	}
	return time.Time{}
}

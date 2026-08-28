package shardadmin

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/dedup"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/dedupkey"
	"ad-event-processor/pkg/regionproxy/quorum"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

type LeaseState string

const (
	LeaseStateBooked    LeaseState = "booked"
	LeaseStateExecuting LeaseState = "executing"
	LeaseStateCompleted LeaseState = "completed"
	LeaseStateExpired   LeaseState = "expired"
)

func (s LeaseState) String() string {
	return string(s)
}

func ValidLeaseState(s string) bool {
	switch LeaseState(s) {
	case LeaseStateBooked, LeaseStateExecuting, LeaseStateCompleted, LeaseStateExpired:
		return true
	default:
		return false
	}
}

type LeaseDedupScope struct {
	RegionID    uuid.UUID `json:"region_id"`
	SourceID    uuid.UUID `json:"source_id"`
	SourceEpoch uint32    `json:"source_epoch"`
	SeqStart    int64     `json:"seq_start"`
	SeqEnd      int64     `json:"seq_end"`
	Attempt     int32     `json:"attempt"`
}

var ErrLeaseRenewExhausted = errors.New("operation lease renew budget exhausted")

type LeaseRenewHook func(opID uuid.UUID)

var ErrLeaseQuorumNotMet = errors.New("operation lease book quorum not met")

var ErrOpKeyPoolShed = errors.New("operation lease book shed: op key pool over watermark")

type OpKeyPoolGate interface {
	ShouldShed() bool
}

const (
	defaultLeaseQuorum     = 2
	opLeaseJanitorLockKey1 = int32(0x4d36)
)

const leaseFencingEpochFile = "fencing.epoch"

var ErrStaleFencingEpoch = errors.New("stale fencing epoch")

type LeaseFencingRegistry struct {
	baseDir string
	mu      sync.Mutex
	stores  map[uuid.UUID]*LeaseFencingStore
}

type LeaseFencingStore struct {
	dir   string
	epoch atomic.Uint64
}

func RelayDeliveryOpID(regionCode uint8, outboxEventID int64) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("ad_event_processor-relay-op:%d:%d", regionCode, outboxEventID)))
}

func ProxyBatchOpID(regionCode uint8, nodeID string, seq uint64, opID uuid.UUID) uuid.UUID {
	if opID != uuid.Nil {
		return opID
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("ad_event_processor-proxy-op:%d:%s:%d", regionCode, nodeID, seq)))
}

func ProxyBatchOpIDFromBytes(opID [16]byte) uuid.UUID {
	var u uuid.UUID
	copy(u[:], opID[:])
	if u == uuid.Nil {
		return uuid.Nil
	}
	return u
}

func EncodeLeaseDedupScope(scope dedupkey.Scope, attempt int32) ([]byte, error) {
	payload := LeaseDedupScope{
		RegionID:    scope.RegionID,
		SourceID:    scope.SourceID,
		SourceEpoch: scope.SourceEpoch,
		SeqStart:    scope.SeqStart,
		SeqEnd:      scope.SeqEnd,
		Attempt:     attempt,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode lease dedup scope: %w", err)
	}
	return raw, nil
}

func DecodeLeaseDedupScope(raw []byte) (dedupkey.Scope, int32, error) {
	var payload LeaseDedupScope
	if err := json.Unmarshal(raw, &payload); err != nil {
		return dedupkey.Scope{}, 0, fmt.Errorf("decode lease dedup scope: %w", err)
	}
	return dedupkey.Scope{
		RegionID:    payload.RegionID,
		SourceID:    payload.SourceID,
		SourceEpoch: payload.SourceEpoch,
		SeqStart:    payload.SeqStart,
		SeqEnd:      payload.SeqEnd,
	}, payload.Attempt, nil
}

func (w *OperationLeaseWorker) renewInterval() time.Duration {
	sec := w.timeoutSec / 3
	if sec < 1 {
		sec = 1
	}
	return time.Duration(sec) * time.Second
}

func (w *OperationLeaseWorker) runRenewHeartbeat(ctx context.Context, opID uuid.UUID, done <-chan struct{}) {
	if w == nil || !w.renewHeartbeat {
		return
	}
	ticker := time.NewTicker(w.renewInterval())
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.RenewLease(ctx, opID); err != nil {
				if errors.Is(err, ErrLeaseRenewExhausted) || errors.Is(err, context.Canceled) {
					return
				}
				slog.Warn("operation lease heartbeat renew failed", "op_id", opID, "error", err)
			}
		}
	}
}

func (w *OperationLeaseWorker) SetRenewHeartbeat(enabled bool) {
	if w == nil {
		return
	}
	w.renewHeartbeat = enabled
}

func (w *OperationLeaseWorker) SetLeaseRenewHook(hook LeaseRenewHook) {
	if w == nil {
		return
	}
	w.onRenew = hook
}


func (w *OperationLeaseWorker) EnsureBook(ctx context.Context, req OperationLeaseBookRequest) (OperationLeaseBookResult, error) {
	empty := OperationLeaseBookResult{}
	if w == nil || w.host == nil || w.host.Pool() == nil {
		return empty, fmt.Errorf("operation lease ensure book op_id=%s: worker unavailable", req.OpID)
	}
	q := db.New(w.host.Pool())
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

func RelayDeliveryBookRequest(ctx context.Context, host LeaseHost, regionCode uint8, outboxEventID int64, eventType string, payload []byte, attempt int32) OperationLeaseBookRequest {
	adapter := host.DedupAdapter(ctx)
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
		ReplicaSetID: uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("ad_event_processor-relay-set:%d", regionCode))),
		Attempt:      attempt,
		FactorU:      factorU,
		Scope:        scope,
		ReplicaNodes: host.LeaseReplicaNodes(),
	}
}

type ProxyBatchBookInput struct {
	RegionCode  uint8
	NodeID      string
	SourceEpoch uint32
	Seq         uint64
	FactorU     uuid.UUID
	OpID        uuid.UUID
}

func ProxyBatchBookRequest(ctx context.Context, host LeaseHost, in ProxyBatchBookInput, attempt int32) OperationLeaseBookRequest {
	epoch := in.SourceEpoch
	if epoch == 0 && host.Pool() != nil {
		epoch = dedup.LoadRoutingEpoch(ctx, host.Pool())
	}
	adapter := dedup.NewAdapter(host.Pool(), in.RegionCode, epoch)
	seq := int64(in.Seq)
	scope := adapter.RegionScope(dedupkey.ProxySourceID(in.RegionCode, in.NodeID), seq, seq)
	if attempt <= 0 {
		attempt = 1
	}
	return OperationLeaseBookRequest{
		OpID:         ProxyBatchOpID(in.RegionCode, in.NodeID, in.Seq, in.OpID),
		RegionCode:   int16(in.RegionCode),
		Role:         "region-proxy",
		ReplicaSetID: uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("ad_event_processor-proxy-set:%d:%s", in.RegionCode, in.NodeID))),
		Attempt:      attempt,
		FactorU:      in.FactorU,
		Scope:        scope,
		ReplicaNodes: []string{in.NodeID},
		BookAckNodes: []string{in.NodeID},
	}
}

func QuorumRequired(replicaCount int) int32 {
	if replicaCount <= 1 {
		return 1
	}
	return defaultLeaseQuorum
}

func ScopeWithAttempt(scope dedupkey.Scope, attempt int32) dedupkey.Scope {
	if attempt < 1 {
		attempt = 1
	}
	if attempt == 1 {
		return scope
	}
	offset := int64(attempt)
	if scope.SeqStart == scope.SeqEnd {
		seq := scope.SeqStart*1000 + offset
		scope.SeqStart = seq
		scope.SeqEnd = seq
		return scope
	}
	scope.SeqStart = scope.SeqStart*1000 + offset
	scope.SeqEnd = scope.SeqEnd*1000 + offset
	return scope
}

func DedupScopeForLease(lease db.OperationLease) (dedupkey.Scope, error) {
	base, attempt, err := DecodeLeaseDedupScope(lease.DedupScope)
	if err != nil {
		return dedupkey.Scope{}, err
	}
	if attempt <= 0 {
		attempt = lease.Attempt
	}
	return ScopeWithAttempt(base, attempt), nil
}

func AuthoritativeLeaseView(lease db.OperationLease, executorNodeID string, now time.Time) error {
	switch LeaseState(lease.LeaseState) {
	case LeaseStateExecuting:
	default:
		opID := uuid.UUID(lease.OpID.Bytes)
		return fmt.Errorf("authoritative lease op_id=%s: state=%s", opID, lease.LeaseState)
	}
	if !lease.DeadlineAt.Valid || !lease.DeadlineAt.Time.After(now) {
		opID := uuid.UUID(lease.OpID.Bytes)
		return fmt.Errorf("authoritative lease op_id=%s: expired deadline", opID)
	}
	if !lease.ExecutorNodeID.Valid || lease.ExecutorNodeID.String != executorNodeID {
		opID := uuid.UUID(lease.OpID.Bytes)
		return fmt.Errorf("authoritative lease op_id=%s: executor mismatch", opID)
	}
	return nil
}

func (w *OperationLeaseWorker) postgresAvailable(ctx context.Context) bool {
	if w == nil || w.host == nil || w.host.Pool() == nil {
		return false
	}
	return w.host.Pool().Ping(ctx) == nil
}

func (w *OperationLeaseWorker) quorumRDB() redis.UniversalClient {
	if w == nil || w.host == nil {
		return nil
	}
	return w.host.ControlRedis()
}

func (w *OperationLeaseWorker) bookRedis(ctx context.Context, req OperationLeaseBookRequest) (OperationLeaseBookResult, error) {
	empty := OperationLeaseBookResult{}
	redisClient := w.quorumRDB()
	if redisClient == nil {
		return empty, ErrLeaseQuorumNotMet
	}
	opID := opIDBytes(req.OpID)
	replicas := req.ReplicaNodes
	if len(replicas) == 0 {
		replicas = []string{w.nodeID}
	}
	for _, n := range req.BookAckNodes {
		st, err := quorum.AckBook(ctx, redisClient, opID, len(replicas), n)
		if err != nil {
			return empty, err
		}
		if st.QuorumMet {
			return operationLeaseBookResultFromQuorum(st), nil
		}
	}
	st, err := quorum.Book(ctx, redisClient, opID, replicas, w.nodeID)
	if err != nil {
		return empty, err
	}
	result := operationLeaseBookResultFromQuorum(st)
	if !result.QuorumMet {
		return result, ErrLeaseQuorumNotMet
	}
	return result, nil
}

func (w *OperationLeaseWorker) ackBookRedis(ctx context.Context, opID uuid.UUID, nodeID string, replicaCount int) (OperationLeaseBookResult, error) {
	empty := OperationLeaseBookResult{}
	redisClient := w.quorumRDB()
	if redisClient == nil {
		return empty, ErrLeaseQuorumNotMet
	}
	st, err := quorum.AckBook(ctx, redisClient, opIDBytes(opID), replicaCount, nodeID)
	if err != nil {
		return empty, err
	}
	return operationLeaseBookResultFromQuorum(st), nil
}

func (w *OperationLeaseWorker) quorumStatusRedis(ctx context.Context, opID uuid.UUID, replicaCount int) (OperationLeaseBookResult, error) {
	redisClient := w.quorumRDB()
	if redisClient == nil {
		return OperationLeaseBookResult{}, ErrLeaseQuorumNotMet
	}
	st, err := quorum.ReadStatus(ctx, redisClient, opIDBytes(opID), replicaCount)
	if err != nil {
		return OperationLeaseBookResult{}, err
	}
	return operationLeaseBookResultFromQuorum(st), nil
}

func operationLeaseBookResultFromQuorum(st quorum.Status) OperationLeaseBookResult {
	return OperationLeaseBookResult{
		AckCount:  st.AckCount,
		Quorum:    st.Quorum,
		QuorumMet: st.QuorumMet,
	}
}

func opIDBytes(id uuid.UUID) [16]byte {
	var out [16]byte
	copy(out[:], id[:])
	return out
}

func isPostgresUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

func NewLeaseFencingRegistry(baseDir string) (*LeaseFencingRegistry, error) {
	if baseDir == "" {
		return nil, errors.New("lease fencing dir required")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	return &LeaseFencingRegistry{
		baseDir: baseDir,
		stores:  make(map[uuid.UUID]*LeaseFencingStore),
	}, nil
}

func (r *LeaseFencingRegistry) storeFor(replicaSetID uuid.UUID) (*LeaseFencingStore, error) {
	if r == nil {
		return nil, errors.New("lease fencing registry unavailable")
	}
	if replicaSetID == uuid.Nil {
		return nil, errors.New("replica set id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.stores[replicaSetID]; ok {
		return s, nil
	}
	dir := filepath.Join(r.baseDir, replicaSetID.String())
	s, err := NewLeaseFencingStore(dir)
	if err != nil {
		return nil, err
	}
	r.stores[replicaSetID] = s
	return s, nil
}

func (r *LeaseFencingRegistry) Next(replicaSetID uuid.UUID) (int64, error) {
	s, err := r.storeFor(replicaSetID)
	if err != nil {
		return 0, err
	}
	return s.Next()
}

func (r *LeaseFencingRegistry) Validate(replicaSetID uuid.UUID, epoch int64) error {
	s, err := r.storeFor(replicaSetID)
	if err != nil {
		return err
	}
	return s.Validate(epoch)
}

func NewLeaseFencingStore(dir string) (*LeaseFencingStore, error) {
	if dir == "" {
		return nil, errors.New("lease fencing dir required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &LeaseFencingStore{dir: dir}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *LeaseFencingStore) Floor() uint64 {
	if s == nil {
		return 0
	}
	return s.epoch.Load()
}

func (s *LeaseFencingStore) Next() (int64, error) {
	if s == nil {
		return 1, nil
	}
	next := s.epoch.Add(1)
	if err := s.persist(next); err != nil {
		return 0, err
	}
	return int64(next), nil
}

func (s *LeaseFencingStore) Validate(epoch int64) error {
	if s == nil {
		return nil
	}
	floor := s.Floor()
	if floor == 0 {
		return nil
	}
	if epoch <= 0 || uint64(epoch) < floor {
		return ErrStaleFencingEpoch
	}
	return nil
}

func (s *LeaseFencingStore) load() error {
	path := filepath.Join(s.dir, leaseFencingEpochFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) < 8 {
		return nil
	}
	s.epoch.Store(binary.BigEndian.Uint64(data[:8]))
	return nil
}

func (s *LeaseFencingStore) persist(epoch uint64) error {
	path := filepath.Join(s.dir, leaseFencingEpochFile)
	tmp := path + ".tmp"
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], epoch)
	if err := os.WriteFile(tmp, buf[:], 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

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

func (w *OperationLeaseWorker) RunJanitor(ctx context.Context) (int32, error) {
	if w == nil || w.host == nil || w.host.Pool() == nil {
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
	err = w.host.WithPostgresLow(ctx, func(runCtx context.Context) error {
		n, err := db.New(w.host.Pool()).OperationLeaseExpireStale(runCtx, defaultOpLeaseExpireBatch)
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
	err := w.host.Pool().QueryRow(ctx,
		`SELECT pg_try_advisory_lock($1::int, $2::int)`,
		opLeaseJanitorLockKey1, int32(w.region),
	).Scan(&ok)
	return ok, err
}

func (w *OperationLeaseWorker) releaseJanitorLock(ctx context.Context) {
	_, _ = w.host.Pool().Exec(ctx,
		`SELECT pg_advisory_unlock($1::int, $2::int)`,
		opLeaseJanitorLockKey1, int32(w.region),
	)
}

func (w *OperationLeaseWorker) ProcessBooked(ctx context.Context) error {
	if w == nil || w.host == nil || w.executor == nil {
		return nil
	}
	if w.opKeyGate != nil && w.opKeyGate.ShouldShed() {
		return nil
	}
	q := db.New(w.host.Pool())
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
			slog.Warn("operation lease execute failed", "op_id", opID, "err", err)
		}
	}
	return nil
}

func (w *OperationLeaseWorker) Start(ctx context.Context) {
	if w == nil || w.host == nil || w.host.Pool() == nil {
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
				slog.Error("operation lease poll failed", "node_id", w.nodeID, "err", err)
			}
		case <-janitorTicker.C:
			if _, err := w.RunJanitor(ctx); err != nil && ctx.Err() == nil {
				slog.Error("operation lease janitor failed", "node_id", w.nodeID, "err", err)
			}
		}
	}
}

func LeaseOpID(lease db.OperationLease) uuid.UUID {
	if lease.OpID.Valid {
		return uuid.UUID(lease.OpID.Bytes)
	}
	return uuid.Nil
}

func LeaseFactorU(lease db.OperationLease) uuid.UUID {
	if lease.FactorU.Valid {
		return uuid.UUID(lease.FactorU.Bytes)
	}
	return uuid.Nil
}

func LeaseDeadline(lease db.OperationLease) time.Time {
	if lease.DeadlineAt.Valid {
		return lease.DeadlineAt.Time
	}
	return time.Time{}
}

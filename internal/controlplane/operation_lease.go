package controlplane

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
	"ad-event-processor/pkg/dedupkey"
	"ad-event-processor/pkg/regionproxy/quorum"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func RelayDeliveryBookRequest(ctx context.Context, s *Service, regionCode uint8, outboxEventID int64, eventType string, payload []byte, attempt int32) OperationLeaseBookRequest {
	adapter := s.dedupAdapter(ctx)
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

func (w *OperationLeaseWorker) pgAvailable(ctx context.Context) bool {
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return false
	}
	return w.svc.pool.Ping(ctx) == nil
}

func (w *OperationLeaseWorker) quorumRDB() redis.UniversalClient {
	if w == nil || w.svc == nil {
		return nil
	}
	return PickHealthyControlShard(w.svc.redisShards)
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

func isPgUnavailable(err error) bool {
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

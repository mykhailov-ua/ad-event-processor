package controlplane

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type QuotaManager struct {
	svc          *Service
	quotaRepo    *domain.QuotaRepo
	pollInterval time.Duration
	chunkSize    int64
	thresholdPct int
}

func NewQuotaManager(svc *Service) *QuotaManager {
	var chunkSize int64
	var thresholdPct int
	if svc.cfg != nil {
		chunkSize = svc.cfg.QuotaChunkSize
		thresholdPct = svc.cfg.QuotaRefillThresholdPct
	}
	if chunkSize <= 0 {
		chunkSize = 5000000
	}
	if thresholdPct <= 0 {
		thresholdPct = 20
	}
	return &QuotaManager{
		svc:          svc,
		quotaRepo:    domain.NewQuotaRepo(svc.GetPool()),
		pollInterval: 100 * time.Millisecond,
		chunkSize:    chunkSize,
		thresholdPct: thresholdPct,
	}
}

func (qm *QuotaManager) Start(ctx context.Context) {
	ticker := time.NewTicker(qm.pollInterval)
	defer ticker.Stop()

	warmTicker := time.NewTicker(5 * time.Second)
	defer warmTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if qm.svc.cfg != nil && (qm.svc.cfg.QuotaMode == "shadow" || qm.svc.cfg.QuotaMode == "live") {
				qm.pollRefills(ctx)
			}
		case <-warmTicker.C:
			if qm.svc.cfg != nil && (qm.svc.cfg.QuotaMode == "shadow" || qm.svc.cfg.QuotaMode == "live") {
				qm.warmActiveCampaignQuotas(ctx)
			}
		}
	}
}

func (qm *QuotaManager) pollRefills(ctx context.Context) {
	for shardIdx, rdb := range qm.svc.rdbs {
		campaignIDs, err := rdb.SPopN(ctx, "budget:refill_needed", 100).Result()
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				slog.Error("failed to pop from budget:refill_needed", "shard", shardIdx, "error", err)
			}
			continue
		}

		for _, idStr := range campaignIDs {
			campaignID, err := uuid.Parse(idStr)
			if err != nil {
				slog.Error("failed to parse campaign ID from refill_needed", "id", idStr, "error", err)
				continue
			}

			if err := qm.refillCampaign(ctx, campaignID, shardIdx, rdb); err != nil {
				slog.Error("failed to refill campaign", "campaign_id", campaignID, "error", err)
			}
		}
	}
}

func (qm *QuotaManager) refillCampaign(ctx context.Context, campaignID uuid.UUID, shardIdx int, rdb redis.UniversalClient) error {
	lockKey := fmt.Sprintf("budget:refill_lock:%s", campaignID)
	claimed, err := rdb.GetDel(ctx, lockKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return fmt.Errorf("claim budget:refill_lock: %w", err)
	}
	if claimed == "" {
		return nil
	}

	requeue := func() {
		_ = rdb.Set(ctx, lockKey, "1", 10*time.Second).Err()
		_ = rdb.SAdd(ctx, "budget:refill_needed", campaignID.String()).Err()
	}

	idempotencyKey := uuid.New().String()
	res, err := qm.quotaRepo.ReserveChunk(ctx, qm.svc.sharder, campaignID, qm.chunkSize, idempotencyKey)
	if err != nil {
		if errors.Is(err, domain.ErrQuotaBudgetExceeded) {
			slog.Warn("campaign budget exceeded during refill", "campaign_id", campaignID)
			return nil
		}
		requeue()
		return fmt.Errorf("failed to reserve chunk in Postgres: %w", err)
	}

	if res.AlreadyApplied {
		return nil
	}

	shadow := qm.svc.cfg != nil && qm.svc.cfg.QuotaMode == "shadow"
	if shadow {
		slog.Info("shadow quota refill reserved in Postgres only",
			"campaign_id", campaignID, "shard", shardIdx, "chunk_size", qm.chunkSize,
			"reserved_amount", res.ReservedAmount)
		return nil
	}

	quotaKey := fmt.Sprintf("budget:quota:%s", campaignID)
	_, err = rdb.IncrBy(ctx, quotaKey, qm.chunkSize).Result()
	if err != nil {
		slog.Error("failed to increment budget:quota in Redis, rolling back Postgres reservation", "campaign_id", campaignID, "error", err)
		if rollbackErr := qm.quotaRepo.ReleaseChunk(ctx, qm.svc.sharder, campaignID, qm.chunkSize); rollbackErr != nil {
			slog.Error("failed to rollback Postgres reservation", "campaign_id", campaignID, "error", rollbackErr)
		}
		requeue()
		return fmt.Errorf("failed to increment budget:quota in Redis: %w", err)
	}

	slog.Info("successfully refilled campaign quota", "campaign_id", campaignID, "shard", shardIdx, "chunk_size", qm.chunkSize)
	return nil
}

func (qm *QuotaManager) warmActiveCampaignQuotas(ctx context.Context) {
	campaignRepo := domain.NewCampaignRepo(db.New(qm.svc.GetPool()))
	campaigns, err := campaignRepo.ListActive(ctx)
	if err != nil {
		slog.Error("failed to list active campaigns for quota warming", "error", err)
		return
	}

	shadow := qm.svc.cfg != nil && qm.svc.cfg.QuotaMode == "shadow"
	byShard := make(map[int][]*domain.Campaign)
	for _, camp := range campaigns {
		if camp == nil {
			continue
		}
		shardIdx := qm.svc.sharder.GetShard(camp.ID)
		if shardIdx < 0 || shardIdx >= len(qm.svc.rdbs) {
			continue
		}
		byShard[shardIdx] = append(byShard[shardIdx], camp)
	}

	for shardIdx, shardCamps := range byShard {
		rdb := qm.svc.rdbs[shardIdx]
		if rdb == nil {
			continue
		}
		campaignIDs := make([]uuid.UUID, len(shardCamps))
		for i, camp := range shardCamps {
			campaignIDs[i] = camp.ID
		}
		existsMap, err := batchQuotaKeyExists(ctx, rdb, campaignIDs)
		if err != nil {
			slog.Error("quota warm: batch exists failed", "shard", shardIdx, "error", err)
			continue
		}

		missing := make([]*domain.Campaign, 0)
		missingIDs := make([]uuid.UUID, 0)
		for _, camp := range shardCamps {
			if existsMap[camp.ID] {
				continue
			}
			missing = append(missing, camp)
			missingIDs = append(missingIDs, camp.ID)
		}
		if len(missing) == 0 {
			continue
		}

		if shadow {
			reservedMap, err := qm.quotaRepo.MapReservedByCampaignIDs(ctx, missingIDs)
			if err != nil {
				slog.Error("quota warm: batch pg lookup failed", "shard", shardIdx, "error", err)
				continue
			}
			filtered := make([]*domain.Campaign, 0, len(missing))
			for _, camp := range missing {
				if reservedMap[camp.ID] > 0 {
					continue
				}
				filtered = append(filtered, camp)
			}
			missing = filtered
		}

		for _, camp := range missing {
			qm.warmCampaignQuota(ctx, shardIdx, rdb, camp.ID, shadow)
		}
	}
}

func (qm *QuotaManager) warmCampaignQuota(ctx context.Context, shardIdx int, rdb redis.UniversalClient, campaignID uuid.UUID, shadow bool) {
	quotaKey := fmt.Sprintf("budget:quota:%s", campaignID)
	lockKey := fmt.Sprintf("budget:refill_lock:%s", campaignID)

	locked, err := rdb.SetNX(ctx, lockKey, "1", 10*time.Second).Result()
	if err != nil || !locked {
		return
	}

	exists, err := rdb.Exists(ctx, quotaKey).Result()
	if err != nil || exists > 0 {
		_ = rdb.Del(ctx, lockKey).Err()
		return
	}

	slog.Info("initializing quota for campaign", "campaign_id", campaignID, "shadow", shadow)
	idempotencyKey := fmt.Sprintf("init-quota-%s", campaignID)
	_, err = qm.quotaRepo.ReserveChunk(ctx, qm.svc.sharder, campaignID, qm.chunkSize, idempotencyKey)
	if err != nil {
		_ = rdb.Del(ctx, lockKey).Err()
		if !errors.Is(err, domain.ErrQuotaBudgetExceeded) {
			slog.Error("failed to reserve initial chunk in Postgres", "campaign_id", campaignID, "error", err)
		}
		return
	}

	actualChunk := qm.chunkSize

	if shadow {
		_ = rdb.Del(ctx, lockKey).Err()
		slog.Info("shadow quota warm reserved in Postgres only",
			"campaign_id", campaignID, "shard", shardIdx, "chunk_size", actualChunk)
		return
	}

	_, err = rdb.IncrBy(ctx, quotaKey, actualChunk).Result()
	if err != nil {
		slog.Error("failed to set initial budget:quota in Redis, rolling back Postgres", "campaign_id", campaignID, "error", err)
		_ = qm.quotaRepo.ReleaseChunk(ctx, qm.svc.sharder, campaignID, actualChunk)
		_ = rdb.Del(ctx, lockKey).Err()
		return
	}

	_ = rdb.Del(ctx, lockKey).Err()
	slog.Info("successfully initialized campaign quota", "campaign_id", campaignID, "shard", shardIdx, "chunk_size", actualChunk)
}

const (
	quotaCrashGapSeconds   = 30
	quotaRepairEventType   = "QUOTA_REPAIR"
	quotaRepairSystemAdmin = "00000000-0000-0000-0000-000000000001"
	quotaRepairTargetType  = "campaign_quota"
)

type QuotaRepairAction string

const (
	QuotaRepairTopUpRedis QuotaRepairAction = "topup_redis"
	QuotaRepairReleasePG  QuotaRepairAction = "release_pg"
)

type QuotaRepairPayload struct {
	CampaignID    string `json:"campaign_id"`
	ShardID       int16  `json:"shard_id"`
	Action        string `json:"action"`
	PgReserved    int64  `json:"pg_reserved"`
	RedisExpected int64  `json:"redis_expected"`
	ChunkSize     int64  `json:"chunk_size"`
	DriftMicro    int64  `json:"drift_micro"`
	RepairMicro   int64  `json:"repair_micro"`
	Reason        string `json:"reason"`
}

type quotaRow struct {
	shardID        int16
	campaignID     uuid.UUID
	reservedAmount int64
	chunkSize      int64
	updatedAt      time.Time
}

func (w *ReconWorker) RepairQuotaDrift(ctx context.Context) {
	if w == nil || w.svc == nil || w.svc.cfg == nil || !w.svc.cfg.QuotaAutoRepair {
		return
	}
	if w.svc.cfg.QuotaMode != "shadow" && w.svc.cfg.QuotaMode != "live" {
		return
	}
	pool := w.svc.GetPool()
	if pool == nil {
		return
	}

	w.observeShardQuorum(ctx)
	w.releaseDeadShardReservations(ctx)

	rows, err := w.loadActiveQuotas(ctx)
	if err != nil {
		slog.Error("quota repair: load active quotas failed", "error", err)
		return
	}

	shardRows := make(map[int][]quotaRow)
	for _, r := range rows {
		if int(r.shardID) >= len(w.svc.rdbs) {
			continue
		}
		if w.quorum != nil && w.quorum.DeadShardConfirmed(int(r.shardID)) {
			continue
		}
		shardRows[int(r.shardID)] = append(shardRows[int(r.shardID)], r)
	}

	for shardID, shardRows := range shardRows {
		rdb := w.svc.rdbs[shardID]
		if rdb == nil {
			continue
		}
		campaignIDs := make([]uuid.UUID, len(shardRows))
		for i, r := range shardRows {
			campaignIDs[i] = r.campaignID
		}
		snapshots, err := batchRedisQuotaExpected(ctx, rdb, campaignIDs)
		if err != nil {
			slog.Error("quota repair: batch redis read failed", "shard", shardID, "error", err)
			continue
		}
		for _, r := range shardRows {
			snap, ok := snapshots[r.campaignID]
			if !ok {
				continue
			}
			action, repairMicro, reason := decideQuotaRepair(r, snap.expected, snap.quotaMissing)
			if action == "" || repairMicro <= 0 {
				continue
			}
			if err := w.enqueueQuotaRepair(ctx, r, action, snap.expected, repairMicro, reason); err != nil {
				slog.Error("quota repair: enqueue failed", "campaign_id", r.campaignID, "error", err)
				continue
			}
			metrics.QuotaRepairEnqueuedTotal.Inc()
		}
	}
}

func (w *ReconWorker) loadActiveQuotas(ctx context.Context) ([]quotaRow, error) {
	p := w.svc.GetPool()
	rows, err := p.Query(ctx, `
		SELECT shard_id, campaign_id, reserved_amount, chunk_size, updated_at
		FROM campaign_quotas
		WHERE reserved_amount > 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []quotaRow
	for rows.Next() {
		var r quotaRow
		var cid uuid.UUID
		if err := rows.Scan(&r.shardID, &cid, &r.reservedAmount, &r.chunkSize, &r.updatedAt); err != nil {
			return nil, err
		}
		r.campaignID = cid
		out = append(out, r)
	}
	return out, rows.Err()
}

type quotaRedisSnapshot struct {
	expected     int64
	quotaMissing bool
}

func batchQuotaKeyExists(ctx context.Context, rdb redis.UniversalClient, campaignIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if rdb == nil || len(campaignIDs) == 0 {
		return map[uuid.UUID]bool{}, nil
	}
	pipe := rdb.Pipeline()
	cmds := make(map[uuid.UUID]*redis.IntCmd, len(campaignIDs))
	for _, id := range campaignIDs {
		cmds[id] = pipe.Exists(ctx, fmt.Sprintf("budget:quota:%s", id))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]bool, len(campaignIDs))
	for id, cmd := range cmds {
		n, err := cmd.Result()
		if err != nil {
			return nil, err
		}
		out[id] = n > 0
	}
	return out, nil
}

func batchRedisQuotaExpected(ctx context.Context, rdb redis.UniversalClient, campaignIDs []uuid.UUID) (map[uuid.UUID]quotaRedisSnapshot, error) {
	if rdb == nil || len(campaignIDs) == 0 {
		return map[uuid.UUID]quotaRedisSnapshot{}, nil
	}
	type quotaCmds struct {
		quota    *redis.StringCmd
		sync     *redis.StringCmd
		inflight *redis.StringCmd
	}
	pipe := rdb.Pipeline()
	cmdByID := make(map[uuid.UUID]quotaCmds, len(campaignIDs))
	for _, id := range campaignIDs {
		cidStr := id.String()
		cmdByID[id] = quotaCmds{
			quota:    pipe.Get(ctx, "budget:quota:"+cidStr),
			sync:     pipe.Get(ctx, "budget:sync:campaign:"+cidStr),
			inflight: pipe.Get(ctx, "budget:inflight:campaign:"+cidStr),
		}
	}
	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	out := make(map[uuid.UUID]quotaRedisSnapshot, len(campaignIDs))
	for id, cmds := range cmdByID {
		quotaVal, qErr := cmds.quota.Int64()
		syncVal, _ := cmds.sync.Int64()
		inflightVal, _ := cmds.inflight.Int64()
		quotaMissing := errors.Is(qErr, redis.Nil)
		if quotaMissing {
			quotaVal = 0
		}
		out[id] = quotaRedisSnapshot{
			expected:     quotaVal + syncVal + inflightVal,
			quotaMissing: quotaMissing,
		}
	}
	return out, nil
}

func decideQuotaRepair(r quotaRow, redisExpected int64, quotaKeyMissing bool) (QuotaRepairAction, int64, string) {
	chunk := r.chunkSize
	if chunk <= 0 {
		chunk = 1
	}
	drift := r.reservedAmount - redisExpected
	absDrift := int64(math.Abs(float64(drift)))

	if quotaKeyMissing && r.reservedAmount > 0 &&
		time.Since(r.updatedAt) >= quotaCrashGapSeconds*time.Second {
		amount := r.reservedAmount
		if amount > chunk {
			amount = chunk
		}
		return QuotaRepairTopUpRedis, amount, "crash_gap_missing_redis_key"
	}

	if absDrift <= chunk {
		return "", 0, ""
	}

	if drift > 0 {
		amount := absDrift
		if amount > chunk*2 {
			amount = chunk
		}
		return QuotaRepairTopUpRedis, amount, "pg_reserved_exceeds_redis"
	}

	return QuotaRepairReleasePG, absDrift, "redis_exceeds_pg_reserved"
}

func (w *ReconWorker) enqueueQuotaRepair(
	ctx context.Context,
	r quotaRow,
	action QuotaRepairAction,
	redisExpected, repairMicro int64,
	reason string,
) error {
	payload, err := coldpath.MarshalOutbox(QuotaRepairPayload{
		CampaignID:    r.campaignID.String(),
		ShardID:       r.shardID,
		Action:        string(action),
		PgReserved:    r.reservedAmount,
		RedisExpected: redisExpected,
		ChunkSize:     r.chunkSize,
		DriftMicro:    r.reservedAmount - redisExpected,
		RepairMicro:   repairMicro,
		Reason:        reason,
	})
	if err != nil {
		return err
	}
	q := db.New(w.svc.GetPool())
	ev, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: quotaRepairEventType,
		Payload:   payload,
	})
	if err != nil {
		return err
	}
	slog.Info("quota repair enqueued",
		"outbox_id", ev.ID,
		"campaign_id", r.campaignID,
		"action", action,
		"repair_micro", repairMicro,
		"reason", reason,
	)
	return nil
}

func (w *ReconWorker) observeShardQuorum(ctx context.Context) {
	if w.quorum == nil {
		return
	}
	for shardIdx, rdb := range w.svc.rdbs {
		w.quorum.ObserveShard(ctx, shardIdx, rdb)
	}
}

func (w *ReconWorker) releaseDeadShardReservations(ctx context.Context) {
	if w.quorum == nil || !w.svc.cfg.QuotaAutoRepair {
		return
	}
	pool := w.svc.GetPool()
	for shardIdx := range w.svc.rdbs {
		if !w.quorum.DeadShardConfirmed(shardIdx) {
			continue
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			continue
		}
		tag, err := tx.Exec(ctx, `
			UPDATE campaign_quotas
			SET reserved_amount = 0, updated_at = NOW()
			WHERE shard_id = $1 AND reserved_amount > 0`,
			int16(shardIdx))
		if err != nil {
			_ = tx.Rollback(ctx)
			slog.Error("dead shard quota release failed", "shard", shardIdx, "error", err)
			continue
		}
		if tag.RowsAffected() == 0 {
			_ = tx.Rollback(ctx)
			continue
		}
		metrics.QuotaDeadShardReleaseTotal.Add(float64(tag.RowsAffected()))
		adminID := uuid.MustParse(quotaRepairSystemAdmin)
		q := db.New(tx)
		w.svc.AuditLog(ctx, q, adminID, "QUOTA_DEAD_SHARD_RELEASE", "redis_shard",
			nil, auditQuotaDeadShardRelease{
				ShardID:      shardIdx,
				RowsReleased: tag.RowsAffected(),
			}, auditTxSourceMeta{TxSource: "recon_worker"})
		if err := tx.Commit(ctx); err != nil {
			slog.Error("dead shard quota release commit failed", "shard", shardIdx, "error", err)
			continue
		}
		slog.Warn("released PG quota reservations for dead shard", "shard", shardIdx, "rows", tag.RowsAffected())
	}
}

func (w *ReconWorker) MonitorQuotaDrift(ctx context.Context) {
	pool := w.svc.GetPool()
	if pool == nil {
		return
	}
	rows, err := w.loadActiveQuotas(ctx)
	if err != nil {
		return
	}
	shardRows := make(map[int][]quotaRow)
	for _, r := range rows {
		if int(r.shardID) >= len(w.svc.rdbs) {
			continue
		}
		shardRows[int(r.shardID)] = append(shardRows[int(r.shardID)], r)
	}
	for shardID, shardRows := range shardRows {
		rdb := w.svc.rdbs[shardID]
		if rdb == nil {
			continue
		}
		pingCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		if err := rdb.Ping(pingCtx).Err(); err != nil {
			cancel()
			continue
		}
		cancel()

		campaignIDs := make([]uuid.UUID, len(shardRows))
		for i, r := range shardRows {
			campaignIDs[i] = r.campaignID
		}
		snapshots, err := batchRedisQuotaExpected(ctx, rdb, campaignIDs)
		if err != nil {
			continue
		}
		for _, r := range shardRows {
			snap, ok := snapshots[r.campaignID]
			if !ok {
				continue
			}
			drift := math.Abs(float64(r.reservedAmount - snap.expected))
			if drift > float64(r.chunkSize) {
				metrics.QuotaDriftDetectedTotal.Inc()
				slog.Error("QUOTA DRIFT DETECTED",
					"campaign_id", r.campaignID,
					"shard", r.shardID,
					"pg_reserved", r.reservedAmount,
					"redis_expected", snap.expected,
					"drift", drift,
					"chunk_size", r.chunkSize,
				)
			}
		}
	}
}

func quotaRepairRedisAppliedKey(outboxEventID int64) string {
	return fmt.Sprintf("quota:repair:redis_applied:%d", outboxEventID)
}

func quotaRepairAuditAction(action QuotaRepairAction) string {
	switch action {
	case QuotaRepairTopUpRedis:
		return "QUOTA_REPAIR_TOPUP"
	case QuotaRepairReleasePG:
		return "QUOTA_REPAIR_RELEASE"
	default:
		return ""
	}
}

func (worker *OutboxWorker) quotaRepairPgAlreadyApplied(ctx context.Context, eventID int64, action QuotaRepairAction, campID uuid.UUID) (bool, error) {
	auditAction := quotaRepairAuditAction(action)
	if auditAction == "" {
		return false, fmt.Errorf("unknown quota repair action: %s", action)
	}
	var exists bool
	err := worker.svc.GetPool().QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM admin_audit_log
			WHERE action = $1 AND target_id = $2
			  AND (metadata->>'outbox_event_id')::bigint = $3
		)`, auditAction, domain.ToUUID(campID), eventID).Scan(&exists)
	return exists, err
}

func (worker *OutboxWorker) ApplyQuotaRepair(ctx context.Context, eventID int64, payload []byte) error {
	p, err := parseQuotaRepairPayload(payload)
	if err != nil {
		return err
	}
	campID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id: %w", err)
	}

	action := QuotaRepairAction(p.Action)
	if err := worker.applyQuotaRepairPG(ctx, eventID, p, campID, action); err != nil {
		return err
	}
	if action == QuotaRepairTopUpRedis {
		if err := worker.applyQuotaRepairRedisTopUp(ctx, eventID, p); err != nil {
			return err
		}
	}
	metrics.QuotaRepairAppliedTotal.Inc()
	return nil
}

func (worker *OutboxWorker) applyQuotaRepairPG(
	ctx context.Context,
	eventID int64,
	p QuotaRepairPayload,
	campID uuid.UUID,
	action QuotaRepairAction,
) error {
	applied, err := worker.quotaRepairPgAlreadyApplied(ctx, eventID, action, campID)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	tx, err := worker.svc.GetPool().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := db.New(tx)
	adminID := uuid.MustParse(quotaRepairSystemAdmin)
	auditMeta := auditQuotaRepairMeta{
		OutboxEventID: eventID,
		Reason:        p.Reason,
		RepairMicro:   p.RepairMicro,
	}

	switch action {
	case QuotaRepairReleasePG:
		if err := q.DecreaseCampaignQuotaReserved(ctx, db.DecreaseCampaignQuotaReservedParams{
			ShardID:        p.ShardID,
			CampaignID:     domain.ToUUID(campID),
			ReservedAmount: p.RepairMicro,
		}); err != nil {
			return err
		}
		worker.svc.AuditLog(ctx, q, adminID, "QUOTA_REPAIR_RELEASE", quotaRepairTargetType,
			&campID, p, auditMeta)
	case QuotaRepairTopUpRedis:
		worker.svc.AuditLog(ctx, q, adminID, "QUOTA_REPAIR_TOPUP", quotaRepairTargetType,
			&campID, p, auditMeta)
	default:
		return fmt.Errorf("unknown quota repair action: %s", p.Action)
	}

	return tx.Commit(ctx)
}

func (worker *OutboxWorker) applyQuotaRepairRedisTopUp(ctx context.Context, eventID int64, p QuotaRepairPayload) error {
	if int(p.ShardID) >= len(worker.svc.rdbs) {
		return fmt.Errorf("invalid shard_id %d", p.ShardID)
	}
	rdb := worker.svc.rdbs[p.ShardID]
	appliedKey := quotaRepairRedisAppliedKey(eventID)

	n, err := rdb.Exists(ctx, appliedKey).Result()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	quotaKey := fmt.Sprintf("budget:quota:%s", p.CampaignID)
	if err := rdb.IncrBy(ctx, quotaKey, p.RepairMicro).Err(); err != nil {
		return err
	}
	return rdb.Set(ctx, appliedKey, "1", 7*24*time.Hour).Err()
}

func parseQuotaRepairPayload(payload []byte) (QuotaRepairPayload, error) {
	p, err := coldpath.UnmarshalStrict[QuotaRepairPayload](payload)
	if err != nil {
		return p, err
	}
	if p.CampaignID == "" || p.RepairMicro <= 0 {
		return p, fmt.Errorf("invalid quota repair payload")
	}
	return p, nil
}

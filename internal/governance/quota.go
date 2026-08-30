package governance

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
	host         Host
	quotaRepo    *domain.QuotaRepo
	pollInterval time.Duration
	chunkSize    int64
	thresholdPct int
}

func NewQuotaManager(host Host) *QuotaManager {
	var chunkSize int64
	var thresholdPct int
	if host.Config() != nil {
		chunkSize = host.Config().QuotaChunkSize
		thresholdPct = host.Config().QuotaRefillThresholdPct
	}
	if chunkSize <= 0 {
		chunkSize = 5000000
	}
	if thresholdPct <= 0 {
		thresholdPct = 20
	}
	return &QuotaManager{
		host:         host,
		quotaRepo:    domain.NewQuotaRepo(host.Pool()),
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
			if qm.host.Config() != nil && (qm.host.Config().QuotaMode == "shadow" || qm.host.Config().QuotaMode == "live") {
				qm.pollRefills(ctx)
			}
		case <-warmTicker.C:
			if qm.host.Config() != nil && (qm.host.Config().QuotaMode == "shadow" || qm.host.Config().QuotaMode == "live") {
				qm.warmActiveCampaignQuotas(ctx)
			}
		}
	}
}

func (qm *QuotaManager) pollRefills(ctx context.Context) {
	for shardIdx, redisClient := range qm.host.RedisShards() {
		campaignIDs, err := redisClient.SPopN(ctx, "budget:refill_needed", 100).Result()
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

			if err := qm.refillCampaign(ctx, campaignID, shardIdx, redisClient); err != nil {
				slog.Error("failed to refill campaign", "campaign_id", campaignID, "error", err)
			}
		}
	}
}

func (qm *QuotaManager) refillCampaign(ctx context.Context, campaignID uuid.UUID, shardIdx int, redisClient redis.UniversalClient) error {
	lockKey := fmt.Sprintf("budget:refill_lock:%s", campaignID)
	claimed, err := redisClient.GetDel(ctx, lockKey).Result()
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
		_ = redisClient.Set(ctx, lockKey, "1", 10*time.Second).Err()
		_ = redisClient.SAdd(ctx, "budget:refill_needed", campaignID.String()).Err()
	}

	idempotencyKey := uuid.New().String()
	res, err := qm.quotaRepo.ReserveChunk(ctx, qm.host.Sharder(), campaignID, qm.chunkSize, idempotencyKey)
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

	shadow := qm.host.Config() != nil && qm.host.Config().QuotaMode == "shadow"
	if shadow {
		slog.Info("shadow quota refill reserved in Postgres only",
			"campaign_id", campaignID, "shard", shardIdx, "chunk_size", qm.chunkSize,
			"reserved_amount", res.ReservedAmount)
		return nil
	}

	quotaKey := fmt.Sprintf("budget:quota:%s", campaignID)
	_, err = redisClient.IncrBy(ctx, quotaKey, qm.chunkSize).Result()
	if err != nil {
		slog.Error("failed to increment budget:quota in Redis, rolling back Postgres reservation", "campaign_id", campaignID, "error", err)
		if rollbackErr := qm.quotaRepo.ReleaseChunk(ctx, qm.host.Sharder(), campaignID, qm.chunkSize); rollbackErr != nil {
			slog.Error("failed to rollback Postgres reservation", "campaign_id", campaignID, "error", rollbackErr)
		}
		requeue()
		return fmt.Errorf("failed to increment budget:quota in Redis: %w", err)
	}

	slog.Info("successfully refilled campaign quota", "campaign_id", campaignID, "shard", shardIdx, "chunk_size", qm.chunkSize)
	return nil
}

func (qm *QuotaManager) warmActiveCampaignQuotas(ctx context.Context) {
	campaignRepo := domain.NewCampaignRepo(db.New(qm.host.Pool()))
	campaigns, err := campaignRepo.ListActive(ctx)
	if err != nil {
		slog.Error("failed to list active campaigns for quota warming", "error", err)
		return
	}

	shadow := qm.host.Config() != nil && qm.host.Config().QuotaMode == "shadow"
	byShard := make(map[int][]*domain.Campaign)
	for _, camp := range campaigns {
		if camp == nil {
			continue
		}
		shardIdx := qm.host.Sharder().GetShard(camp.ID)
		if shardIdx < 0 || shardIdx >= len(qm.host.RedisShards()) {
			continue
		}
		byShard[shardIdx] = append(byShard[shardIdx], camp)
	}

	for shardIdx, shardCamps := range byShard {
		redisClient := qm.host.RedisShards()[shardIdx]
		if redisClient == nil {
			continue
		}
		campaignIDs := make([]uuid.UUID, len(shardCamps))
		for i, camp := range shardCamps {
			campaignIDs[i] = camp.ID
		}
		existsMap, err := batchQuotaKeyExists(ctx, redisClient, campaignIDs)
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
			qm.warmCampaignQuota(ctx, shardIdx, redisClient, camp.ID, shadow)
		}
	}
}

func (qm *QuotaManager) warmCampaignQuota(ctx context.Context, shardIdx int, redisClient redis.UniversalClient, campaignID uuid.UUID, shadow bool) {
	quotaKey := fmt.Sprintf("budget:quota:%s", campaignID)
	lockKey := fmt.Sprintf("budget:refill_lock:%s", campaignID)

	locked, err := redisClient.SetNX(ctx, lockKey, "1", 10*time.Second).Result()
	if err != nil || !locked {
		return
	}

	exists, err := redisClient.Exists(ctx, quotaKey).Result()
	if err != nil || exists > 0 {
		_ = redisClient.Del(ctx, lockKey).Err()
		return
	}

	slog.Info("initializing quota for campaign", "campaign_id", campaignID, "shadow", shadow)
	idempotencyKey := fmt.Sprintf("init-quota-%s", campaignID)
	_, err = qm.quotaRepo.ReserveChunk(ctx, qm.host.Sharder(), campaignID, qm.chunkSize, idempotencyKey)
	if err != nil {
		_ = redisClient.Del(ctx, lockKey).Err()
		if !errors.Is(err, domain.ErrQuotaBudgetExceeded) {
			slog.Error("failed to reserve initial chunk in Postgres", "campaign_id", campaignID, "error", err)
		}
		return
	}

	actualChunk := qm.chunkSize

	if shadow {
		_ = redisClient.Del(ctx, lockKey).Err()
		slog.Info("shadow quota warm reserved in Postgres only",
			"campaign_id", campaignID, "shard", shardIdx, "chunk_size", actualChunk)
		return
	}

	_, err = redisClient.IncrBy(ctx, quotaKey, actualChunk).Result()
	if err != nil {
		slog.Error("failed to set initial budget:quota in Redis, rolling back Postgres", "campaign_id", campaignID, "error", err)
		_ = qm.quotaRepo.ReleaseChunk(ctx, qm.host.Sharder(), campaignID, actualChunk)
		_ = redisClient.Del(ctx, lockKey).Err()
		return
	}

	_ = redisClient.Del(ctx, lockKey).Err()
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
	CampaignID       string `json:"campaign_id"`
	ShardID          int16  `json:"shard_id"`
	Action           string `json:"action"`
	PostgresReserved int64  `json:"pg_reserved"`
	RedisExpected    int64  `json:"redis_expected"`
	ChunkSize        int64  `json:"chunk_size"`
	DriftMicro       int64  `json:"drift_micro"`
	RepairMicro      int64  `json:"repair_micro"`
	Reason           string `json:"reason"`
}

type quotaRow struct {
	shardID        int16
	campaignID     uuid.UUID
	reservedAmount int64
	chunkSize      int64
	updatedAt      time.Time
}

type QuorumTracker interface {
	ObserveShard(ctx context.Context, shard int, redisClient redis.UniversalClient)
	DeadShardConfirmed(shard int) bool
}

type QuotaRepairRunner struct {
	host   Host
	quorum QuorumTracker
}

func NewQuotaRepairRunner(host Host, quorum QuorumTracker) *QuotaRepairRunner {
	return &QuotaRepairRunner{host: host, quorum: quorum}
}

func (r *QuotaRepairRunner) ReconcileQuotas(ctx context.Context) {
	cfg := r.host.Config()
	if cfg == nil || (cfg.QuotaMode != "shadow" && cfg.QuotaMode != "live") {
		return
	}
	if cfg.QuotaAutoRepair {
		r.RepairQuotaDrift(ctx)
	} else {
		r.MonitorQuotaDrift(ctx)
	}
}

func (r *QuotaRepairRunner) RepairQuotaDrift(ctx context.Context) {
	if r == nil || r.host == nil || r.host.Config() == nil || !r.host.Config().QuotaAutoRepair {
		return
	}
	if r.host.Config().QuotaMode != "shadow" && r.host.Config().QuotaMode != "live" {
		return
	}
	pool := r.host.Pool()
	if pool == nil {
		return
	}

	r.observeShardQuorum(ctx)
	r.releaseDeadShardReservations(ctx)

	rows, err := r.loadActiveQuotas(ctx)
	if err != nil {
		slog.Error("quota repair: load active quotas failed", "error", err)
		return
	}

	shardRows := make(map[int][]quotaRow)
	redisShards := r.host.RedisShards()
	for _, row := range rows {
		if int(row.shardID) >= len(redisShards) {
			continue
		}
		if r.quorum != nil && r.quorum.DeadShardConfirmed(int(row.shardID)) {
			continue
		}
		shardRows[int(row.shardID)] = append(shardRows[int(row.shardID)], row)
	}

	for shardID, shardRows := range shardRows {
		redisClient := redisShards[shardID]
		if redisClient == nil {
			continue
		}
		campaignIDs := make([]uuid.UUID, len(shardRows))
		for i, row := range shardRows {
			campaignIDs[i] = row.campaignID
		}
		snapshots, err := batchRedisQuotaExpected(ctx, redisClient, campaignIDs)
		if err != nil {
			slog.Error("quota repair: batch redis read failed", "shard", shardID, "error", err)
			continue
		}
		for _, row := range shardRows {
			snap, ok := snapshots[row.campaignID]
			if !ok {
				continue
			}
			action, repairMicro, reason := decideQuotaRepair(row, snap.expected, snap.quotaMissing)
			if action == "" || repairMicro <= 0 {
				continue
			}
			if err := r.enqueueQuotaRepair(ctx, row, action, snap.expected, repairMicro, reason); err != nil {
				slog.Error("quota repair: enqueue failed", "campaign_id", row.campaignID, "error", err)
				continue
			}
			metrics.QuotaRepairEnqueuedTotal.Inc()
		}
	}
}

func (r *QuotaRepairRunner) loadActiveQuotas(ctx context.Context) ([]quotaRow, error) {
	p := r.host.Pool()
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

func batchQuotaKeyExists(ctx context.Context, redisClient redis.UniversalClient, campaignIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if redisClient == nil || len(campaignIDs) == 0 {
		return map[uuid.UUID]bool{}, nil
	}
	pipe := redisClient.Pipeline()
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

func batchRedisQuotaExpected(ctx context.Context, redisClient redis.UniversalClient, campaignIDs []uuid.UUID) (map[uuid.UUID]quotaRedisSnapshot, error) {
	if redisClient == nil || len(campaignIDs) == 0 {
		return map[uuid.UUID]quotaRedisSnapshot{}, nil
	}
	type quotaCmds struct {
		quota    *redis.StringCmd
		sync     *redis.StringCmd
		inflight *redis.StringCmd
	}
	pipe := redisClient.Pipeline()
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
		return QuotaRepairTopUpRedis, amount, "crash_window_missing_redis_key"
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

func (r *QuotaRepairRunner) enqueueQuotaRepair(
	ctx context.Context,
	row quotaRow,
	action QuotaRepairAction,
	redisExpected, repairMicro int64,
	reason string,
) error {
	payload, err := coldpath.MarshalOutbox(QuotaRepairPayload{
		CampaignID:       row.campaignID.String(),
		ShardID:          row.shardID,
		Action:           string(action),
		PostgresReserved: row.reservedAmount,
		RedisExpected:    redisExpected,
		ChunkSize:        row.chunkSize,
		DriftMicro:       row.reservedAmount - redisExpected,
		RepairMicro:      repairMicro,
		Reason:           reason,
	})
	if err != nil {
		return err
	}
	q := db.New(r.host.Pool())
	ev, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: quotaRepairEventType,
		Payload:   payload,
	})
	if err != nil {
		return err
	}
	slog.Info("quota repair enqueued",
		"outbox_id", ev.ID,
		"campaign_id", row.campaignID,
		"action", action,
		"repair_micro", repairMicro,
		"reason", reason,
	)
	return nil
}

func (r *QuotaRepairRunner) observeShardQuorum(ctx context.Context) {
	if r.quorum == nil {
		return
	}
	for shardIdx, redisClient := range r.host.RedisShards() {
		r.quorum.ObserveShard(ctx, shardIdx, redisClient)
	}
}

func (r *QuotaRepairRunner) releaseDeadShardReservations(ctx context.Context) {
	if r.quorum == nil || !r.host.Config().QuotaAutoRepair {
		return
	}
	pool := r.host.Pool()
	for shardIdx := range r.host.RedisShards() {
		if !r.quorum.DeadShardConfirmed(shardIdx) {
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
		r.host.AuditLog(ctx, q, adminID, "QUOTA_DEAD_SHARD_RELEASE", "redis_shard",
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

func (r *QuotaRepairRunner) MonitorQuotaDrift(ctx context.Context) {
	pool := r.host.Pool()
	if pool == nil {
		return
	}
	rows, err := r.loadActiveQuotas(ctx)
	if err != nil {
		return
	}
	redisShards := r.host.RedisShards()
	shardRows := make(map[int][]quotaRow)
	for _, row := range rows {
		if int(row.shardID) >= len(redisShards) {
			continue
		}
		shardRows[int(row.shardID)] = append(shardRows[int(row.shardID)], row)
	}
	for shardID, shardRows := range shardRows {
		redisClient := redisShards[shardID]
		if redisClient == nil {
			continue
		}
		pingCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		if err := redisClient.Ping(pingCtx).Err(); err != nil {
			cancel()
			continue
		}
		cancel()

		campaignIDs := make([]uuid.UUID, len(shardRows))
		for i, r := range shardRows {
			campaignIDs[i] = r.campaignID
		}
		snapshots, err := batchRedisQuotaExpected(ctx, redisClient, campaignIDs)
		if err != nil {
			continue
		}
		for _, row := range shardRows {
			snap, ok := snapshots[row.campaignID]
			if !ok {
				continue
			}
			drift := math.Abs(float64(row.reservedAmount - snap.expected))
			if drift > float64(row.chunkSize) {
				metrics.QuotaDriftDetectedTotal.Inc()
				slog.Error("QUOTA DRIFT DETECTED",
					"campaign_id", row.campaignID,
					"shard", row.shardID,
					"pg_reserved", row.reservedAmount,
					"redis_expected", snap.expected,
					"drift", drift,
					"chunk_size", row.chunkSize,
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

func (w *OutboxWorker) quotaRepairPostgresAlreadyApplied(ctx context.Context, eventID int64, action QuotaRepairAction, campID uuid.UUID) (bool, error) {
	auditAction := quotaRepairAuditAction(action)
	if auditAction == "" {
		return false, fmt.Errorf("unknown quota repair action: %s", action)
	}
	var exists bool
	err := w.host.Pool().QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM admin_audit_log
			WHERE action = $1 AND target_id = $2
			 AND (metadata->>'outbox_event_id')::bigint = $3
		)`, auditAction, domain.ToUUID(campID), eventID).Scan(&exists)
	return exists, err
}

func (w *OutboxWorker) ApplyQuotaRepair(ctx context.Context, eventID int64, payload []byte) error {
	p, err := parseQuotaRepairPayload(payload)
	if err != nil {
		return err
	}
	campID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id: %w", err)
	}

	action := QuotaRepairAction(p.Action)
	if err := w.applyQuotaRepairPostgres(ctx, eventID, p, campID, action); err != nil {
		return err
	}
	if action == QuotaRepairTopUpRedis {
		if err := w.applyQuotaRepairRedisTopUp(ctx, eventID, p); err != nil {
			return err
		}
	}
	metrics.QuotaRepairAppliedTotal.Inc()
	return nil
}

func (w *OutboxWorker) applyQuotaRepairPostgres(
	ctx context.Context,
	eventID int64,
	p QuotaRepairPayload,
	campID uuid.UUID,
	action QuotaRepairAction,
) error {
	applied, err := w.quotaRepairPostgresAlreadyApplied(ctx, eventID, action, campID)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	tx, err := w.host.Pool().Begin(ctx)
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
		w.host.AuditLog(ctx, q, adminID, "QUOTA_REPAIR_RELEASE", quotaRepairTargetType,
			&campID, p, auditMeta)
	case QuotaRepairTopUpRedis:
		w.host.AuditLog(ctx, q, adminID, "QUOTA_REPAIR_TOPUP", quotaRepairTargetType,
			&campID, p, auditMeta)
	default:
		return fmt.Errorf("unknown quota repair action: %s", p.Action)
	}

	return tx.Commit(ctx)
}

func (w *OutboxWorker) applyQuotaRepairRedisTopUp(ctx context.Context, eventID int64, p QuotaRepairPayload) error {
	shards := w.host.RedisShards()
	if int(p.ShardID) >= len(shards) {
		return fmt.Errorf("invalid shard_id %d", p.ShardID)
	}
	redisClient := shards[p.ShardID]
	appliedKey := quotaRepairRedisAppliedKey(eventID)

	n, err := redisClient.Exists(ctx, appliedKey).Result()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	quotaKey := fmt.Sprintf("budget:quota:%s", p.CampaignID)
	if err := redisClient.IncrBy(ctx, quotaKey, p.RepairMicro).Err(); err != nil {
		return err
	}
	return redisClient.Set(ctx, appliedKey, "1", 7*24*time.Hour).Err()
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

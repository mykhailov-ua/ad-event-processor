package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"espx/internal/database"
	"espx/internal/domain"
	db "espx/internal/domain/db"
	"espx/internal/metrics"
	"espx/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type campaignIDPayload struct {
	CampaignID string `json:"campaign_id"`
}

type brandIDPayload struct {
	BrandID string `json:"brand_id"`
}

type brandFcapOutboxPayload struct {
	BrandID    string `json:"brand_id"`
	FreqLimit  int32  `json:"freq_limit"`
	FreqWindow int32  `json:"freq_window"`
}

type campaignScheduleOutboxPayload struct {
	CampaignID   string     `json:"campaign_id"`
	StartAt      *time.Time `json:"start_at,omitempty"`
	EndAt        *time.Time `json:"end_at,omitempty"`
	DaypartHours []int16    `json:"daypart_hours,omitempty"`
}

type campaignPacingPayload struct {
	CampaignID string `json:"campaign_id"`
	PacingMode string `json:"pacing_mode"`
}

func (worker *OutboxWorker) handleOutboxEvent(opCtx, ctx context.Context, ev db.OutboxEvent) error {
	switch ev.EventType {
	case "CREATE_CAMPAIGN":
		return worker.handleCreateCampaign(ctx, ev.Payload)
	case "PAUSE_CAMPAIGN":
		return worker.handlePauseCampaign(ctx, ev.Payload)
	case "BUDGET_FREEZE":
		return worker.handleBudgetFreeze(ctx, ev.Payload)
	case "QUOTA_REPAIR":
		return worker.ApplyQuotaRepair(ctx, ev.ID, ev.Payload)
	case "RECONCILIATION_ADJUST":
		return worker.ApplyReconciliationAdjust(ctx, ev.ID, ev.Payload)
	case "RESUME_CAMPAIGN":
		return worker.handleResumeCampaign(ctx, ev.Payload)
	case "UPDATE_CAMPAIGN_SCHEDULE":
		return worker.handleUpdateCampaignSchedule(ctx, ev.Payload)
	case "SYNC_BRAND_CREATIVES":
		return worker.handleSyncBrandCreatives(ctx, ev.Payload)
	case "CANCEL_CAMPAIGN":
		return worker.handleCancelCampaign(ctx, ev.Payload)
	case "UPDATE_CAMPAIGN_PACING":
		return worker.handleUpdateCampaignPacing(ctx, ev.Payload)
	case "UPDATE_CAMPAIGN_FRAUD":
		return worker.handleUpdateCampaignFraud(ctx, ev.Payload)
	case "UPDATE_SETTINGS":
		return worker.handleUpdateSettings(opCtx, ev.ID, ev.Payload)
	case "UPDATE_BLACKLIST":
		return worker.handleUpdateBlacklist(ctx, ev.Payload, ev.CreatedAt.Time)
	case "CONFIGURE_BRAND_FCAP":
		return worker.handleConfigureBrandFcap(ctx, ev.Payload)
	case "UPDATE_SUPPLY_FILES":
		return worker.handleUpdateSupplyFiles(ctx, ev.Payload)
	case "RELOAD_RTB_CATALOG":
		return worker.handleReloadRtbCatalog(ctx, ev.Payload)
	case "SYNC_USER_CONSENT":
		return worker.handleSyncUserConsent(ctx, ev.Payload)
	case "UPDATE_CAMPAIGN_CONSENT":
		return worker.handleUpdateCampaignConsent(ctx, ev.Payload)
	case "UPDATE_COHORT_SNAPSHOT":
		return worker.handleUpdateCohortSnapshot(ctx)
	case "PURGE_USER_DATA":
		return worker.handlePurgeUserData(ctx, ev.Payload)
	case "ML_SCORE_BOOST":
		return worker.handleFraudScoreBoost(ctx, ev.Payload)
	case "ML_GHOST_IVT":
		return worker.handleFraudGhostIVT(ctx, ev.Payload)
	case "ML_BLACKLIST_ADD":
		return worker.handleFraudBlacklistAdd(ctx, ev.Payload)
	case "ML_MODEL_VERSION":
		return worker.handleFraudModelVersion(ctx, ev.Payload)
	case "PAUSE_PLACEMENT":
		return worker.handlePausePlacement(ctx, ev.Payload)
	case "UPDATE_ENTITLEMENTS":
		return worker.handleUpdateEntitlements(ctx)
	case "APPLY_GTV_SETTLEMENT":
		return worker.handleApplyGTVSettlement(ctx, ev.Payload)
	case "TELEGRAM_EVENT":
		return worker.handleTelegramEvent(ctx, ev.Payload)
	default:
		return fmt.Errorf("unknown outbox event type: %s", ev.EventType)
	}
}

func (worker *OutboxWorker) handleCreateCampaign(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[CampaignPayload](payload)
	if err != nil {
		return err
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	rdb := worker.svc.getRDB(campUUID)
	if rdb == nil {
		return fmt.Errorf("no redis client available")
	}
	_, err = rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		return worker.setCampaignBudgetRemaining(ctx, pipe, p.CampaignID, campUUID, p.BudgetLimit)
	})
	if err != nil {
		return err
	}
	return worker.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (worker *OutboxWorker) handlePauseCampaign(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[CampaignPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	return worker.deleteCampaignBudgetAndPublish(ctx, p.CampaignID, campUUID)
}

func (worker *OutboxWorker) handleBudgetFreeze(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[CampaignPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	rdb := worker.svc.getRDB(campUUID)
	if rdb == nil {
		return nil
	}
	if err := domain.SetBudgetFrozen(ctx, rdb, campUUID); err != nil {
		return err
	}
	return worker.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (worker *OutboxWorker) handleResumeCampaign(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[CampaignPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	return worker.setCampaignBudgetAndPublish(ctx, p, campUUID)
}

func (worker *OutboxWorker) handleUpdateCampaignSchedule(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[campaignIDPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	return worker.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (worker *OutboxWorker) handleUpdateCampaignFraud(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[campaignIDPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	return worker.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (worker *OutboxWorker) handleSyncBrandCreatives(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[brandIDPayload](payload)
	if p.BrandID == "" {
		return nil
	}
	return worker.syncBrandCreativesToRedis(ctx, p.BrandID)
}

func (worker *OutboxWorker) handleCancelCampaign(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[CampaignPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	return worker.deleteCampaignBudgetAndPublish(ctx, p.CampaignID, campUUID)
}

func (worker *OutboxWorker) handleUpdateCampaignPacing(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[campaignPacingPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	rdb := worker.svc.getRDB(campUUID)
	if rdb == nil {
		return nil
	}
	_, err = rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, fmt.Sprintf("campaign:settings:%s", p.CampaignID), "pacing_mode", p.PacingMode)
		return nil
	})
	if err != nil {
		return err
	}
	return worker.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (worker *OutboxWorker) handleUpdateSettings(opCtx context.Context, eventID int64, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[SettingsPayload](payload)
	if err != nil {
		return err
	}
	if len(worker.svc.rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	return syncGlobalConfigToAllShards(opCtx, worker.svc.rdbs, p.Settings, eventID)
}

func (worker *OutboxWorker) handleUpdateBlacklist(ctx context.Context, payload []byte, queuedAt time.Time) error {
	p, err := coldpath.UnmarshalStrict[BlacklistPayload](payload)
	if err != nil {
		return err
	}
	return worker.applyBlacklistPayload(ctx, p, queuedAt)
}

func (worker *OutboxWorker) handleConfigureBrandFcap(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[brandFcapOutboxPayload](payload)
	if err != nil {
		return err
	}
	brandUUID, err := uuid.Parse(p.BrandID)
	if err != nil {
		return err
	}
	campIDs, err := worker.listActiveCampaignIDsByBrand(ctx, brandUUID)
	if err != nil {
		return err
	}
	if len(campIDs) == 0 {
		return nil
	}
	channel := worker.svc.campaignUpdateChannel()
	return publishControlMessagesToAllShards(ctx, worker.svc.rdbs, channel, campIDs)
}

func (worker *OutboxWorker) listActiveCampaignIDsByBrand(ctx context.Context, brandUUID uuid.UUID) ([]string, error) {
	rows, err := worker.svc.GetPool().Query(ctx, "SELECT id FROM campaigns WHERE brand_id = $1 AND status = 'ACTIVE'", ToUUID(brandUUID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var campIDs []string
	for rows.Next() {
		var cid uuid.UUID
		if scanErr := rows.Scan(&cid); scanErr == nil {
			campIDs = append(campIDs, cid.String())
		}
	}
	return campIDs, nil
}

func (worker *OutboxWorker) setCampaignBudgetAndPublish(ctx context.Context, p CampaignPayload, campUUID uuid.UUID) error {
	rdb := worker.svc.getRDB(campUUID)
	if rdb == nil {
		return nil
	}
	_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		return worker.setCampaignBudgetRemaining(ctx, pipe, p.CampaignID, campUUID, p.BudgetLimit)
	})
	if err != nil {
		return err
	}
	return worker.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (worker *OutboxWorker) deleteCampaignBudgetAndPublish(ctx context.Context, campaignIDStr string, campUUID uuid.UUID) error {
	rdb := worker.svc.getRDB(campUUID)
	if rdb == nil {
		return nil
	}
	_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, fmt.Sprintf("budget:campaign:%s", campaignIDStr))
		return nil
	})
	if err != nil {
		return err
	}
	return worker.svc.publishCampaignUpdate(ctx, campaignIDStr)
}

type userConsentOutboxPayload struct {
	UserIDHash string `json:"user_id_hash"`
	Purposes   int16  `json:"purposes"`
}

type purgeUserDataPayload struct {
	ErasureID     string `json:"erasure_id"`
	UserIDHash    string `json:"user_id_hash"`
	SubjectUserID string `json:"subject_user_id"`
}

func (worker *OutboxWorker) handleSyncUserConsent(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[userConsentOutboxPayload](payload)
	if err != nil {
		return err
	}
	return worker.svc.SyncUserConsentToRedis(ctx, p.UserIDHash, p.Purposes)
}

func (worker *OutboxWorker) handleUpdateCampaignConsent(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[campaignIDPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	return worker.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (worker *OutboxWorker) handlePurgeUserData(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[purgeUserDataPayload](payload)
	if err != nil {
		return err
	}
	erasureID, err := uuid.Parse(p.ErasureID)
	if err != nil {
		return err
	}
	purgeErr := worker.svc.PurgeUserDataRedis(ctx, p.UserIDHash, p.SubjectUserID)
	return worker.svc.MarkErasureRedisPurgeDone(ctx, erasureID, purgeErr)
}

func (worker *OutboxWorker) handleFraudScoreBoost(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[FraudThreatPayload](payload)
	if err != nil {
		return err
	}
	if p.CampaignID == "" {
		return nil
	}
	if len(worker.svc.rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}

	key := fmt.Sprintf("ml:score:boost:%s", p.CampaignID)
	if p.Boost <= 0 || p.TTLSeconds <= 0 {
		if err := deleteGlobalKeyFromAllShards(ctx, worker.svc.rdbs, key); err != nil {
			return err
		}
	} else {
		ttl := time.Duration(p.TTLSeconds) * time.Second
		boostStr := strconv.Itoa(int(p.Boost))
		if err := syncGlobalStringToAllShards(ctx, worker.svc.rdbs, key, boostStr, ttl); err != nil {
			return err
		}
	}

	return worker.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (worker *OutboxWorker) handleFraudGhostIVT(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[FraudThreatPayload](payload)
	if err != nil {
		return err
	}
	if p.CampaignID == "" {
		return nil
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id: %w", err)
	}

	_, err = worker.svc.GetPool().Exec(ctx, "UPDATE campaigns SET ghost_ivt_enabled = TRUE WHERE id = $1", ToUUID(campUUID))
	if err != nil {
		return fmt.Errorf("failed to update ghost_ivt_enabled: %w", err)
	}

	return worker.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (worker *OutboxWorker) handleFraudBlacklistAdd(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[FraudThreatPayload](payload)
	if err != nil {
		return err
	}
	if p.IP == "" {
		return nil
	}
	ttl := p.TTLSeconds
	_, err = worker.svc.blockIPWithTTL(ctx, p.IP, "fraud", &ttl, false)
	return err
}

func (worker *OutboxWorker) handleFraudModelVersion(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[FraudModelVersionPayload](payload)
	if err != nil {
		return err
	}

	if len(worker.svc.rdbs) == 0 {
		return fmt.Errorf("no redis shards configured")
	}

	writeToShard := func(shardID int) error {
		rdb := worker.svc.rdbs[shardID]
		if rdb == nil {
			return nil
		}
		if err := rdb.Set(ctx, "ml:model:version", p.ModelVersion, 0).Err(); err != nil {
			return fmt.Errorf("failed to set ml:model:version on shard %d: %w", shardID, err)
		}
		if err := rdb.Set(ctx, "ml:model:hash", p.Hash, 0).Err(); err != nil {
			return fmt.Errorf("failed to set ml:model:hash on shard %d: %w", shardID, err)
		}
		if err := rdb.Set(ctx, "ml:model:applied_at", time.Now().Unix(), 0).Err(); err != nil {
			return fmt.Errorf("failed to set ml:model:applied_at on shard %d: %w", shardID, err)
		}
		return nil
	}

	if p.ShardID >= 0 && p.ShardID < len(worker.svc.rdbs) {
		return writeToShard(p.ShardID)
	}

	for i := range worker.svc.rdbs {
		if err := writeToShard(i); err != nil {
			return err
		}
	}

	return nil
}

func (worker *OutboxWorker) handlePausePlacement(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[PausePlacementPayload](payload)
	if p.CampaignID == "" || p.PlacementID == "" {
		return nil
	}
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id: %w", err)
	}

	key := domain.PlacementBlacklistKey(uuid.MustParse(p.CampaignID))
	del := p.Action == "remove"
	return syncGlobalHashFieldToAllShards(ctx, worker.svc.rdbs, key, p.PlacementID, "1", del)
}

type PausePlacementPayload struct {
	CampaignID  string `json:"campaign_id"`
	PlacementID string `json:"placement_id"`
	Action      string `json:"action,omitempty"`
}

func (worker *OutboxWorker) handleUpdateCohortSnapshot(ctx context.Context) error {
	if worker == nil || worker.svc == nil {
		return fmt.Errorf("service unavailable")
	}
	return worker.svc.publishRegistryFullSync(ctx)
}

func (worker *OutboxWorker) handleUpdateEntitlements(ctx context.Context) error {
	if worker == nil || worker.svc == nil {
		return fmt.Errorf("service unavailable")
	}
	return worker.svc.publishRegistryFullSync(ctx)
}

func (worker *OutboxWorker) handleReloadRtbCatalog(ctx context.Context, payload []byte) error {
	_ = payload
	return worker.svc.PublishRtbCatalogReload(ctx)
}

const (
	outboxPollActiveInterval = 20 * time.Millisecond
	outboxPollIdleMax        = 250 * time.Millisecond
)

type outboxPollBackoff struct {
	idle time.Duration
}

func newOutboxPollBackoff() *outboxPollBackoff {
	return &outboxPollBackoff{idle: outboxPollActiveInterval}
}

func (b *outboxPollBackoff) next(processed int) time.Duration {
	if processed > 0 {
		b.idle = outboxPollActiveInterval
		metrics.OutboxPollIntervalMs.Observe(float64(outboxPollActiveInterval) / float64(time.Millisecond))
		return 0
	}
	if b.idle < outboxPollActiveInterval {
		b.idle = outboxPollActiveInterval
	}
	next := b.idle * 2
	if next > outboxPollIdleMax {
		next = outboxPollIdleMax
	}
	b.idle = next
	metrics.OutboxPollIntervalMs.Observe(float64(next) / float64(time.Millisecond))
	return next
}

func (worker *OutboxWorker) recordOutboxLagMetrics(ctx context.Context) {
	if worker.svc == nil || worker.svc.GetPool() == nil {
		return
	}
	opCtx, cancel := workerContext(ctx, workerOutboxTimeout)
	defer cancel()

	var pending int64
	var oldestSeconds float64
	err := worker.svc.GetPool().QueryRow(opCtx, `
		SELECT COUNT(*)::bigint,
		       COALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(created_at))), 0)::float8
		FROM outbox_events
		WHERE status = 'PENDING'`).Scan(&pending, &oldestSeconds)
	if err != nil {
		if ctx.Err() != nil || database.IsShutdownError(err) {
			return
		}
		return
	}
	metrics.ManagementOutboxPendingTotal.Set(float64(pending))
	metrics.ManagementOutboxOldestPendingSeconds.Set(oldestSeconds)

	if worker.svc != nil && worker.svc.alerter != nil && pending > 0 {
		threshold := float64(worker.svc.alerter.OutboxStuckThresholdSec())
		if oldestSeconds >= threshold {
			worker.svc.alerter.AlertOutboxStuck(pending, oldestSeconds)
		}
	}
}

func (worker *OutboxWorker) recordOutboxLagFromValues(pending int64, oldestSeconds float64) {
	metrics.ManagementOutboxPendingTotal.Set(float64(pending))
	metrics.ManagementOutboxOldestPendingSeconds.Set(oldestSeconds)
	if worker.svc != nil && worker.svc.alerter != nil && pending > 0 {
		threshold := float64(worker.svc.alerter.OutboxStuckThresholdSec())
		if oldestSeconds >= threshold {
			worker.svc.alerter.AlertOutboxStuck(pending, oldestSeconds)
		}
	}
}

type telegramEventPayload struct {
	CampaignID uuid.UUID `json:"campaign_id"`
	BotID      int64     `json:"bot_id"`
	Payload    []byte    `json:"payload"`
}

func (worker *OutboxWorker) handleTelegramEvent(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[telegramEventPayload](payload)
	if err != nil {
		return err
	}

	var update struct {
		UpdateID int64 `json:"update_id"`
		Message  *struct {
			Chat struct {
				ID   int64  `json:"id"`
				Type string `json:"type"`
			} `json:"chat"`
			Text string `json:"text"`
			From *struct {
				ID        int64 `json:"id"`
				IsPremium bool  `json:"is_premium"`
			} `json:"from"`
		} `json:"message"`
	}
	if err := json.Unmarshal(p.Payload, &update); err != nil {
		return err
	}

	if update.Message == nil || update.Message.Text == "" {
		return nil
	}

	text := strings.TrimSpace(update.Message.Text)
	if !strings.HasPrefix(text, "/start ") {
		return nil
	}
	token := strings.TrimSpace(strings.TrimPrefix(text, "/start"))

	if token == "" {
		return nil
	}

	q := db.New(worker.svc.GetPool())
	deeplink, err := q.GetTelegramDeeplink(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	if time.Now().After(deeplink.ExpiresAt.Time) {
		_ = q.DeleteTelegramDeeplink(ctx, token)
		return nil
	}

	bot, err := q.GetTelegramBotByBotID(ctx, p.BotID)
	if err != nil {
		return err
	}

	isGroup := update.Message.Chat.Type == "group" || update.Message.Chat.Type == "supergroup" || update.Message.Chat.Type == "channel"

	tgSvc := NewTelegramService(worker.svc, worker.svc.GetPool(), worker.svc.RedisShards())
	err = tgSvc.limiter.Wait(ctx, update.Message.Chat.ID, isGroup)
	if err != nil {
		return err
	}

	welcomeMsg := fmt.Sprintf("Welcome! Click here to start the app: %s", bot.WebhookUrl)

	err = tgSvc.sendBotMessage(ctx, bot.BotToken, update.Message.Chat.ID, welcomeMsg)
	if err != nil {
		var apiErr *tgAPIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 429 {
			if apiErr.RetryAfter > 0 {
				tgSvc.limiter.BackoffChat(update.Message.Chat.ID, time.Duration(apiErr.RetryAfter)*time.Second)
			}
		}
		return err
	}

	return nil
}

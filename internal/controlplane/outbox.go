package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/coldpath"

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

func (w *OutboxWorker) handleOutboxEvent(opCtx, ctx context.Context, ev db.OutboxEvent) error {
	switch ev.EventType {
	case "CREATE_CAMPAIGN":
		return w.handleCreateCampaign(ctx, ev.Payload)
	case "PAUSE_CAMPAIGN":
		return w.handlePauseCampaign(ctx, ev.Payload)
	case "BUDGET_FREEZE":
		return w.handleBudgetFreeze(ctx, ev.Payload)
	case "QUOTA_REPAIR":
		return w.ApplyQuotaRepair(ctx, ev.ID, ev.Payload)
	case "RECONCILIATION_ADJUST":
		return w.ApplyReconciliationAdjust(ctx, ev.ID, ev.Payload)
	case "RESUME_CAMPAIGN":
		return w.handleResumeCampaign(ctx, ev.Payload)
	case "UPDATE_CAMPAIGN_SCHEDULE":
		return w.handleUpdateCampaignSchedule(ctx, ev.Payload)
	case "SYNC_BRAND_CREATIVES":
		return w.handleSyncBrandCreatives(ctx, ev.Payload)
	case "CANCEL_CAMPAIGN":
		return w.handleCancelCampaign(ctx, ev.Payload)
	case "UPDATE_CAMPAIGN_PACING":
		return w.handleUpdateCampaignPacing(ctx, ev.Payload)
	case "UPDATE_CAMPAIGN_FRAUD":
		return w.handleUpdateCampaignFraud(ctx, ev.Payload)
	case "UPDATE_SETTINGS":
		return w.handleUpdateSettings(opCtx, ev.ID, ev.Payload)
	case "UPDATE_BLACKLIST":
		return w.handleUpdateBlacklist(ctx, ev.Payload, ev.CreatedAt.Time)
	case "CONFIGURE_BRAND_FCAP":
		return w.handleConfigureBrandFcap(ctx, ev.Payload)
	case "UPDATE_SUPPLY_FILES":
		return w.handleUpdateSupplyFiles(ctx, ev.Payload)
	case "RELOAD_RTB_CATALOG":
		return w.handleReloadRtbCatalog(ctx, ev.Payload)
	case "SYNC_USER_CONSENT":
		return w.handleSyncUserConsent(ctx, ev.Payload)
	case "UPDATE_CAMPAIGN_CONSENT":
		return w.handleUpdateCampaignConsent(ctx, ev.Payload)
	case "UPDATE_COHORT_SNAPSHOT":
		return w.handleUpdateCohortSnapshot(ctx)
	case "PURGE_USER_DATA":
		return w.handlePurgeUserData(ctx, ev.Payload)
	case "ML_SCORE_BOOST":
		return w.handleFraudScoreBoost(ctx, ev.Payload)
	case "ML_SILENT_REJECT", "ML_GHOST_IVT":
		return w.handleFraudSilentReject(ctx, ev.Payload)
	case "ML_BLACKLIST_ADD":
		return w.handleFraudBlacklistAdd(ctx, ev.Payload)
	case "ML_MODEL_VERSION":
		return w.handleFraudModelVersion(ctx, ev.Payload)
	case "PAUSE_PLACEMENT":
		return w.handlePausePlacement(ctx, ev.Payload)
	case "UPDATE_ENTITLEMENTS":
		return w.handleUpdateEntitlements(ctx)
	case "APPLY_GTV_SETTLEMENT":
		return w.handleApplyGTVSettlement(ctx, ev.Payload)
	case "TELEGRAM_EVENT":
		return w.handleTelegramEvent(ctx, ev.Payload)
	case "LANDER_PUBLISHED":
		return w.handleLanderPublished(ctx, ev.Payload)
	default:
		return fmt.Errorf("unknown outbox event type: %s", ev.EventType)
	}
}

func (w *OutboxWorker) handleCreateCampaign(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[CampaignPayload](payload)
	if err != nil {
		return err
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	redisClient := w.svc.redisClientForCampaign(campUUID)
	if redisClient == nil {
		return fmt.Errorf("no redis client available")
	}
	_, err = redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		return w.setCampaignBudgetRemaining(ctx, pipe, p.CampaignID, campUUID, p.BudgetLimit)
	})
	if err != nil {
		return err
	}
	return w.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (w *OutboxWorker) handlePauseCampaign(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[CampaignPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	return w.deleteCampaignBudgetAndPublish(ctx, p.CampaignID, campUUID)
}

func (w *OutboxWorker) handleBudgetFreeze(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[CampaignPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	redisClient := w.svc.redisClientForCampaign(campUUID)
	if redisClient == nil {
		return nil
	}
	if err := domain.SetBudgetFrozen(ctx, redisClient, campUUID); err != nil {
		return err
	}
	return w.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (w *OutboxWorker) handleResumeCampaign(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[CampaignPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	return w.setCampaignBudgetAndPublish(ctx, p, campUUID)
}

func (w *OutboxWorker) handleUpdateCampaignSchedule(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[campaignIDPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	return w.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (w *OutboxWorker) handleUpdateCampaignFraud(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[campaignIDPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	return w.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (w *OutboxWorker) handleSyncBrandCreatives(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[brandIDPayload](payload)
	if p.BrandID == "" {
		return nil
	}
	return w.syncBrandCreativesToRedis(ctx, p.BrandID)
}

func (w *OutboxWorker) handleCancelCampaign(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[CampaignPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	return w.deleteCampaignBudgetAndPublish(ctx, p.CampaignID, campUUID)
}

func (w *OutboxWorker) handleUpdateCampaignPacing(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[campaignPacingPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	redisClient := w.svc.redisClientForCampaign(campUUID)
	if redisClient == nil {
		return nil
	}
	_, err = redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, fmt.Sprintf("campaign:settings:%s", p.CampaignID), "pacing_mode", p.PacingMode)
		return nil
	})
	if err != nil {
		return err
	}
	return w.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (w *OutboxWorker) handleUpdateSettings(opCtx context.Context, eventID int64, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[SettingsPayload](payload)
	if err != nil {
		return err
	}
	if len(w.svc.redisShards) == 0 {
		return fmt.Errorf("no redis client available")
	}
	return syncGlobalConfigToAllShards(opCtx, w.svc.redisShards, p.Settings, eventID)
}

func (w *OutboxWorker) handleUpdateBlacklist(ctx context.Context, payload []byte, queuedAt time.Time) error {
	p, err := coldpath.UnmarshalStrict[BlacklistPayload](payload)
	if err != nil {
		return err
	}
	return w.applyBlacklistPayload(ctx, p, queuedAt)
}

func (w *OutboxWorker) handleConfigureBrandFcap(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[brandFcapOutboxPayload](payload)
	if err != nil {
		return err
	}
	brandUUID, err := uuid.Parse(p.BrandID)
	if err != nil {
		return err
	}
	campIDs, err := w.listActiveCampaignIDsByBrand(ctx, brandUUID)
	if err != nil {
		return err
	}
	if len(campIDs) == 0 {
		return nil
	}
	channel := w.svc.campaignUpdateChannel()
	return publishControlMessagesToAllShards(ctx, w.svc.redisShards, channel, campIDs)
}

func (w *OutboxWorker) listActiveCampaignIDsByBrand(ctx context.Context, brandUUID uuid.UUID) ([]string, error) {
	rows, err := w.svc.GetPool().Query(ctx, "SELECT id FROM campaigns WHERE brand_id = $1 AND status = 'ACTIVE'", ToUUID(brandUUID))
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

func (w *OutboxWorker) setCampaignBudgetAndPublish(ctx context.Context, p CampaignPayload, campUUID uuid.UUID) error {
	redisClient := w.svc.redisClientForCampaign(campUUID)
	if redisClient == nil {
		return nil
	}
	_, err := redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		return w.setCampaignBudgetRemaining(ctx, pipe, p.CampaignID, campUUID, p.BudgetLimit)
	})
	if err != nil {
		return err
	}
	return w.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (w *OutboxWorker) deleteCampaignBudgetAndPublish(ctx context.Context, campaignIDStr string, campUUID uuid.UUID) error {
	redisClient := w.svc.redisClientForCampaign(campUUID)
	if redisClient == nil {
		return nil
	}
	_, err := redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, fmt.Sprintf("budget:campaign:%s", campaignIDStr))
		return nil
	})
	if err != nil {
		return err
	}
	return w.svc.publishCampaignUpdate(ctx, campaignIDStr)
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

func (w *OutboxWorker) handleSyncUserConsent(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[userConsentOutboxPayload](payload)
	if err != nil {
		return err
	}
	return w.svc.SyncUserConsentToRedis(ctx, p.UserIDHash, p.Purposes)
}

func (w *OutboxWorker) handleUpdateCampaignConsent(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[campaignIDPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	return w.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (w *OutboxWorker) handlePurgeUserData(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[purgeUserDataPayload](payload)
	if err != nil {
		return err
	}
	erasureID, err := uuid.Parse(p.ErasureID)
	if err != nil {
		return err
	}
	purgeErr := w.svc.PurgeUserDataRedis(ctx, p.UserIDHash, p.SubjectUserID)
	return w.svc.MarkErasureRedisPurgeDone(ctx, erasureID, purgeErr)
}

func (w *OutboxWorker) handleFraudScoreBoost(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[FraudThreatPayload](payload)
	if err != nil {
		return err
	}
	if p.CampaignID == "" {
		return nil
	}
	if len(w.svc.redisShards) == 0 {
		return fmt.Errorf("no redis client available")
	}

	key := fmt.Sprintf("ml:score:boost:%s", p.CampaignID)
	if p.Boost <= 0 || p.TTLSeconds <= 0 {
		if err := deleteGlobalKeyFromAllShards(ctx, w.svc.redisShards, key); err != nil {
			return err
		}
	} else {
		ttl := time.Duration(p.TTLSeconds) * time.Second
		boostStr := strconv.Itoa(int(p.Boost))
		if err := syncGlobalStringToAllShards(ctx, w.svc.redisShards, key, boostStr, ttl); err != nil {
			return err
		}
	}

	return w.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (w *OutboxWorker) handleFraudSilentReject(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[FraudThreatPayload](payload)
	if err != nil {
		return err
	}
	if p.IP == "" {
		return nil
	}
	return w.applyMLBlacklistSingle(ctx, payload)
}

func (w *OutboxWorker) handleFraudBlacklistAdd(ctx context.Context, payload []byte) error {
	return w.applyMLBlacklistSingle(ctx, payload)
}

func (w *OutboxWorker) handleFraudModelVersion(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[FraudModelVersionPayload](payload)
	if err != nil {
		return err
	}

	if len(w.svc.redisShards) == 0 {
		return fmt.Errorf("no redis shards configured")
	}

	writeToShard := func(shardID int) error {
		redisClient := w.svc.redisShards[shardID]
		if redisClient == nil {
			return nil
		}
		if err := redisClient.Set(ctx, "ml:model:version", p.ModelVersion, 0).Err(); err != nil {
			return fmt.Errorf("failed to set ml:model:version on shard %d: %w", shardID, err)
		}
		if err := redisClient.Set(ctx, "ml:model:hash", p.Hash, 0).Err(); err != nil {
			return fmt.Errorf("failed to set ml:model:hash on shard %d: %w", shardID, err)
		}
		if err := redisClient.Set(ctx, "ml:model:applied_at", time.Now().Unix(), 0).Err(); err != nil {
			return fmt.Errorf("failed to set ml:model:applied_at on shard %d: %w", shardID, err)
		}
		return nil
	}

	if p.ShardID >= 0 && p.ShardID < len(w.svc.redisShards) {
		return writeToShard(p.ShardID)
	}

	for i := range w.svc.redisShards {
		if err := writeToShard(i); err != nil {
			return err
		}
	}

	return nil
}

func (w *OutboxWorker) handlePausePlacement(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[PausePlacementPayload](payload)
	if p.CampaignID == "" || p.PlacementID == "" {
		return nil
	}
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id: %w", err)
	}

	key := domain.PlacementBlacklistKey(uuid.MustParse(p.CampaignID))
	del := p.Action == "remove"
	return syncGlobalHashFieldToAllShards(ctx, w.svc.redisShards, key, p.PlacementID, "1", del)
}

type PausePlacementPayload struct {
	CampaignID  string `json:"campaign_id"`
	PlacementID string `json:"placement_id"`
	Action      string `json:"action,omitempty"`
}

func (w *OutboxWorker) handleUpdateCohortSnapshot(ctx context.Context) error {
	if w == nil || w.svc == nil {
		return fmt.Errorf("service unavailable")
	}
	return w.svc.publishRegistryFullSync(ctx)
}

func (w *OutboxWorker) handleUpdateEntitlements(ctx context.Context) error {
	if w == nil || w.svc == nil {
		return fmt.Errorf("service unavailable")
	}
	return w.svc.publishRegistryFullSync(ctx)
}

func (w *OutboxWorker) handleReloadRtbCatalog(ctx context.Context, payload []byte) error {
	_ = payload
	return w.svc.PublishRtbCatalogReload(ctx)
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

func (w *OutboxWorker) recordOutboxLagMetrics(ctx context.Context) {
	if w.svc == nil || w.svc.GetPool() == nil {
		return
	}
	opCtx, cancel := workerContext(ctx, workerOutboxTimeout)
	defer cancel()

	var pending int64
	var oldestSeconds float64
	err := w.svc.GetPool().QueryRow(opCtx, `
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
	metrics.SetControlOutboxQueueMetrics(pending, oldestSeconds)

	if w.svc != nil && w.svc.alerter != nil && pending > 0 {
		threshold := float64(w.svc.alerter.OutboxStuckThresholdSec())
		if oldestSeconds >= threshold {
			w.svc.alerter.AlertOutboxStuck(ctx, pending, oldestSeconds)
		}
	}
}

func (w *OutboxWorker) recordOutboxLagFromValues(ctx context.Context, pending int64, oldestSeconds float64) {
	metrics.SetControlOutboxQueueMetrics(pending, oldestSeconds)
	if w.svc != nil && w.svc.alerter != nil && pending > 0 {
		threshold := float64(w.svc.alerter.OutboxStuckThresholdSec())
		if oldestSeconds >= threshold {
			w.svc.alerter.AlertOutboxStuck(ctx, pending, oldestSeconds)
		}
	}
}

type telegramEventPayload struct {
	CampaignID uuid.UUID `json:"campaign_id"`
	BotID      int64     `json:"bot_id"`
	Payload    []byte    `json:"payload"`
}

func (w *OutboxWorker) handleTelegramEvent(ctx context.Context, payload []byte) error {
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

	q := db.New(w.svc.GetPool())
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

	tgSvc := NewTelegramService(w.svc, w.svc.GetPool(), w.svc.RedisShards())

	if update.Message.From != nil {
		_ = tgSvc.recordWebhookStartEvent(
			ctx,
			FromUUID(deeplink.CampaignID),
			p.BotID,
			token,
			update.Message.From.ID,
			update.Message.Chat.Type,
			update.Message.From.IsPremium,
		)
		relayCtx, relayCancel := telegramPostbackRelayContext(ctx)
		tgSvc.relayPostbacks(relayCtx, FromUUID(deeplink.CampaignID), token)
		relayCancel()
	}

	err = tgSvc.limiter.Wait(ctx, update.Message.Chat.ID, isGroup)
	if err != nil {
		return err
	}

	landingURL := bot.MiniAppUrl
	if landingURL == "" {
		landingURL = bot.WebhookUrl
	}
	welcomeMsg := fmt.Sprintf("Welcome! Click here to start the app: %s", landingURL)

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

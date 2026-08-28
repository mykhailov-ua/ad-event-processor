package outbox

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (w *Worker) handleOutboxEvent(opCtx, ctx context.Context, ev db.OutboxEvent) error {
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

func (w *Worker) handleCreateCampaign(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[CampaignPayload](payload)
	if err != nil {
		return err
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	redisClient := w.host.RedisClientForCampaign(campUUID)
	if redisClient == nil {
		return fmt.Errorf("no redis client available")
	}
	_, err = redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		return w.setCampaignBudgetRemaining(ctx, pipe, p.CampaignID, campUUID, p.BudgetLimit)
	})
	if err != nil {
		return err
	}
	w.host.PublishCampaignUpdate(ctx, p.CampaignID)
	return nil
}

func (w *Worker) handlePauseCampaign(ctx context.Context, payload []byte) error {
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

func (w *Worker) handleBudgetFreeze(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[CampaignPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	redisClient := w.host.RedisClientForCampaign(campUUID)
	if redisClient == nil {
		return nil
	}
	if err := domain.SetBudgetFrozen(ctx, redisClient, campUUID); err != nil {
		return err
	}
	w.host.PublishCampaignUpdate(ctx, p.CampaignID)
	return nil
}

func (w *Worker) handleResumeCampaign(ctx context.Context, payload []byte) error {
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

func (w *Worker) handleUpdateCampaignSchedule(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[CampaignIDPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	w.host.PublishCampaignUpdate(ctx, p.CampaignID)
	return nil
}

func (w *Worker) handleUpdateCampaignFraud(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[CampaignIDPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	w.host.PublishCampaignUpdate(ctx, p.CampaignID)
	return nil
}

func (w *Worker) handleSyncBrandCreatives(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[BrandIDPayload](payload)
	if p.BrandID == "" {
		return nil
	}
	return w.syncBrandCreativesToRedis(ctx, p.BrandID)
}

func (w *Worker) handleCancelCampaign(ctx context.Context, payload []byte) error {
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

func (w *Worker) handleUpdateCampaignPacing(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[CampaignPacingPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	campUUID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	redisClient := w.host.RedisClientForCampaign(campUUID)
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
	w.host.PublishCampaignUpdate(ctx, p.CampaignID)
	return nil
}

func (w *Worker) handleUpdateSettings(opCtx context.Context, eventID int64, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[SettingsPayload](payload)
	if err != nil {
		return err
	}
	if len(w.host.RedisShards()) == 0 {
		return fmt.Errorf("no redis client available")
	}
	return SyncGlobalConfigToAllShards(opCtx, w.host.RedisShards(), p.Settings, eventID)
}

func (w *Worker) handleUpdateBlacklist(ctx context.Context, payload []byte, queuedAt time.Time) error {
	p, err := coldpath.UnmarshalStrict[BlacklistPayload](payload)
	if err != nil {
		return err
	}
	return w.applyBlacklistPayload(ctx, p, queuedAt)
}

func (w *Worker) handleConfigureBrandFcap(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[BrandFcapPayload](payload)
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
	channel := w.host.CampaignUpdateChannel()
	return PublishControlMessagesToAllShards(ctx, w.host.RedisShards(), channel, campIDs)
}

func (w *Worker) listActiveCampaignIDsByBrand(ctx context.Context, brandUUID uuid.UUID) ([]string, error) {
	rows, err := w.host.Pool().Query(ctx, "SELECT id FROM campaigns WHERE brand_id = $1 AND status = 'ACTIVE'", domain.ToUUID(brandUUID))
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

func (w *Worker) setCampaignBudgetAndPublish(ctx context.Context, p CampaignPayload, campUUID uuid.UUID) error {
	redisClient := w.host.RedisClientForCampaign(campUUID)
	if redisClient == nil {
		return nil
	}
	_, err := redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		return w.setCampaignBudgetRemaining(ctx, pipe, p.CampaignID, campUUID, p.BudgetLimit)
	})
	if err != nil {
		return err
	}
	w.host.PublishCampaignUpdate(ctx, p.CampaignID)
	return nil
}

func (w *Worker) deleteCampaignBudgetAndPublish(ctx context.Context, campaignIDStr string, campUUID uuid.UUID) error {
	redisClient := w.host.RedisClientForCampaign(campUUID)
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
	w.host.PublishCampaignUpdate(ctx, campaignIDStr)
	return nil
}

func (w *Worker) handleSyncUserConsent(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[UserConsentPayload](payload)
	if err != nil {
		return err
	}
	return w.host.SyncUserConsentToRedis(ctx, p.UserIDHash, p.Purposes)
}

func (w *Worker) handleUpdateCampaignConsent(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[CampaignIDPayload](payload)
	if p.CampaignID == "" {
		return nil
	}
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id in payload: %w", err)
	}
	w.host.PublishCampaignUpdate(ctx, p.CampaignID)
	return nil
}

func (w *Worker) handlePurgeUserData(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[PurgeUserDataPayload](payload)
	if err != nil {
		return err
	}
	erasureID, err := uuid.Parse(p.ErasureID)
	if err != nil {
		return err
	}
	purgeErr := w.host.PurgeUserDataRedis(ctx, p.UserIDHash, p.SubjectUserID)
	return w.host.MarkErasureRedisPurgeDone(ctx, erasureID, purgeErr)
}

func (w *Worker) handleFraudScoreBoost(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[FraudThreatPayload](payload)
	if err != nil {
		return err
	}
	if p.CampaignID == "" {
		return nil
	}
	if len(w.host.RedisShards()) == 0 {
		return fmt.Errorf("no redis client available")
	}

	key := fmt.Sprintf("ml:score:boost:%s", p.CampaignID)
	if p.Boost <= 0 || p.TTLSeconds <= 0 {
		if err := DeleteGlobalKeyFromAllShards(ctx, w.host.RedisShards(), key); err != nil {
			return err
		}
	} else {
		ttl := time.Duration(p.TTLSeconds) * time.Second
		boostStr := strconv.Itoa(int(p.Boost))
		if err := SyncGlobalStringToAllShards(ctx, w.host.RedisShards(), key, boostStr, ttl); err != nil {
			return err
		}
	}

	w.host.PublishCampaignUpdate(ctx, p.CampaignID)
	return nil
}

func (w *Worker) handleFraudSilentReject(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[FraudThreatPayload](payload)
	if err != nil {
		return err
	}
	if p.IP == "" {
		return nil
	}
	return w.applyMLBlacklistSingle(ctx, payload)
}

func (w *Worker) handleFraudBlacklistAdd(ctx context.Context, payload []byte) error {
	return w.applyMLBlacklistSingle(ctx, payload)
}

func (w *Worker) handleFraudModelVersion(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[FraudModelVersionPayload](payload)
	if err != nil {
		return err
	}

	if len(w.host.RedisShards()) == 0 {
		return fmt.Errorf("no redis shards configured")
	}

	writeToShard := func(shardID int) error {
		redisClient := w.host.RedisShards()[shardID]
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

	if p.ShardID >= 0 && p.ShardID < len(w.host.RedisShards()) {
		return writeToShard(p.ShardID)
	}

	for i := range w.host.RedisShards() {
		if err := writeToShard(i); err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) handlePausePlacement(ctx context.Context, payload []byte) error {
	p := coldpath.UnmarshalLenient[PausePlacementPayload](payload)
	if p.CampaignID == "" || p.PlacementID == "" {
		return nil
	}
	if _, err := uuid.Parse(p.CampaignID); err != nil {
		return fmt.Errorf("invalid campaign id: %w", err)
	}

	key := domain.PlacementBlacklistKey(uuid.MustParse(p.CampaignID))
	del := p.Action == "remove"
	return SyncGlobalHashFieldToAllShards(ctx, w.host.RedisShards(), key, p.PlacementID, "1", del)
}

func (w *Worker) handleUpdateCohortSnapshot(ctx context.Context) error {
	if w == nil || w.host == nil {
		return fmt.Errorf("service unavailable")
	}
	return w.host.PublishRegistryFullSync(ctx)
}

func (w *Worker) handleUpdateEntitlements(ctx context.Context) error {
	if w == nil || w.host == nil {
		return fmt.Errorf("service unavailable")
	}
	return w.host.PublishRegistryFullSync(ctx)
}

func (w *Worker) handleReloadRtbCatalog(ctx context.Context, payload []byte) error {
	_ = payload
	return w.host.PublishRtbCatalogReload(ctx)
}

func (w *Worker) handleTelegramEvent(ctx context.Context, payload []byte) error {
	return w.host.HandleTelegramEvent(ctx, payload)
}


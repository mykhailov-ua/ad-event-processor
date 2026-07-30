package management

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"espx/internal/ingestion"
	db "espx/internal/ingestion/sqlc"
	"espx/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type campaignIDPayload struct {
	CampaignID string `json:"campaign_id"`
}

type brandIDPayload struct {
	BrandID string `json:"brand_id"`
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
	case "ML_GHOST_IVT":
		return w.handleFraudGhostIVT(ctx, ev.Payload)
	case "ML_BLACKLIST_ADD":
		return w.handleFraudBlacklistAdd(ctx, ev.Payload)
	case "ML_MODEL_VERSION":
		return w.handleFraudModelVersion(ctx, ev.Payload)
	case "PAUSE_PLACEMENT":
		return w.handlePausePlacement(ctx, ev.Payload)
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
	rdb := w.svc.getRDB(campUUID)
	if rdb == nil {
		return fmt.Errorf("no redis client available")
	}
	_, err = rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
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
	rdb := w.svc.getRDB(campUUID)
	if rdb == nil {
		return nil
	}
	if err := ingestion.SetBudgetFrozen(ctx, rdb, campUUID); err != nil {
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
	rdb := w.svc.getRDB(campUUID)
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
	return w.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (w *OutboxWorker) handleUpdateSettings(opCtx context.Context, eventID int64, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[SettingsPayload](payload)
	if err != nil {
		return err
	}
	if len(w.svc.rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	return syncGlobalConfigToAllShards(opCtx, w.svc.rdbs, p.Settings, eventID)
}

func (w *OutboxWorker) handleUpdateBlacklist(ctx context.Context, payload []byte, queuedAt time.Time) error {
	p, err := coldpath.UnmarshalStrict[BlacklistPayload](payload)
	if err != nil {
		return err
	}
	return w.applyBlacklistPayload(ctx, p, queuedAt)
}

func (w *OutboxWorker) handleConfigureBrandFcap(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[brandIDPayload](payload)
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
	rdb := w.svc.getPubSubRDB()
	if rdb == nil {
		return nil
	}
	channel := w.svc.campaignUpdateChannel()
	_, err = rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, cidStr := range campIDs {
			pipe.Publish(ctx, channel, cidStr)
		}
		return nil
	})
	return err
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
	rdb := w.svc.getRDB(campUUID)
	if rdb == nil {
		return nil
	}
	_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		return w.setCampaignBudgetRemaining(ctx, pipe, p.CampaignID, campUUID, p.BudgetLimit)
	})
	if err != nil {
		return err
	}
	return w.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (w *OutboxWorker) deleteCampaignBudgetAndPublish(ctx context.Context, campaignIDStr string, campUUID uuid.UUID) error {
	rdb := w.svc.getRDB(campUUID)
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
	if len(w.svc.rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}

	key := fmt.Sprintf("ml:score:boost:%s", p.CampaignID)
	if p.Boost <= 0 || p.TTLSeconds <= 0 {
		if err := deleteGlobalKeyFromAllShards(ctx, w.svc.rdbs, key); err != nil {
			return err
		}
	} else {
		ttl := time.Duration(p.TTLSeconds) * time.Second
		boostStr := strconv.Itoa(int(p.Boost))
		if err := syncGlobalStringToAllShards(ctx, w.svc.rdbs, key, boostStr, ttl); err != nil {
			return err
		}
	}

	return w.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (w *OutboxWorker) handleFraudGhostIVT(ctx context.Context, payload []byte) error {
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

	_, err = w.svc.GetPool().Exec(ctx, "UPDATE campaigns SET ghost_ivt_enabled = TRUE WHERE id = $1", ToUUID(campUUID))
	if err != nil {
		return fmt.Errorf("failed to update ghost_ivt_enabled: %w", err)
	}

	return w.svc.publishCampaignUpdate(ctx, p.CampaignID)
}

func (w *OutboxWorker) handleFraudBlacklistAdd(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[FraudThreatPayload](payload)
	if err != nil {
		return err
	}
	if p.IP == "" {
		return nil
	}
	ttl := p.TTLSeconds
	_, err = w.svc.blockIPWithTTL(ctx, p.IP, "fraud", &ttl, false)
	return err
}

func (w *OutboxWorker) handleFraudModelVersion(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[FraudModelVersionPayload](payload)
	if err != nil {
		return err
	}

	if len(w.svc.rdbs) == 0 {
		return fmt.Errorf("no redis shards configured")
	}

	writeToShard := func(shardID int) error {
		rdb := w.svc.rdbs[shardID]
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

	if p.ShardID >= 0 && p.ShardID < len(w.svc.rdbs) {
		return writeToShard(p.ShardID)
	}

	for i := range w.svc.rdbs {
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

	key := ingestion.PlacementBlacklistKey(uuid.MustParse(p.CampaignID))
	del := p.Action == "remove"
	return syncGlobalHashFieldToAllShards(ctx, w.svc.rdbs, key, p.PlacementID, "1", del)
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

package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type CampaignFraudConfigUpdate struct {
	Preset                *string `json:"preset,omitempty"`
	FraudThresholdPass    *uint8  `json:"fraud_threshold_pass,omitempty"`
	FraudThresholdSuspect *uint8  `json:"fraud_threshold_suspect,omitempty"`
	FraudThresholdIVT     *uint8  `json:"fraud_threshold_ivt,omitempty"`
	FraudThresholdBlock   *uint8  `json:"fraud_threshold_block,omitempty"`
	SilentRejectEnabled   *bool   `json:"silent_reject_enabled,omitempty"`
	BehaviorFlags         *uint32 `json:"behavior_flags,omitempty"`
}

func validateFraudThresholds(pass, suspect, ivt, block uint8) error {
	if pass > 100 || suspect > 100 || ivt > 100 || block > 100 {
		return errValidation("fraud thresholds must be between 0 and 100")
	}
	if pass > suspect || suspect > ivt || ivt > block {
		return errValidation("fraud thresholds must be ordered: pass <= suspect <= ivt <= block")
	}
	return nil
}

func (s *Service) GetCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID) (CampaignFraudConfigDTO, error) {
	row, err := db.New(s.GetPool()).GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return CampaignFraudConfigDTO{}, mapNotFound(err, ErrCampaignNotFound)
	}
	return CampaignFraudConfigDTO{
		CampaignID:            campaignID.String(),
		FraudThresholdPass:    uint8(row.FraudThresholdPass),
		FraudThresholdSuspect: uint8(row.FraudThresholdSuspect),
		FraudThresholdIVT:     uint8(row.FraudThresholdIvt),
		FraudThresholdBlock:   uint8(row.FraudThresholdBlock),
		SilentRejectEnabled:   row.SilentRejectEnabled,
		BehaviorFlags:         uint32(row.BehaviorFlags),
	}, nil
}

func (s *Service) UpdateCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID, upd CampaignFraudConfigUpdate) (CampaignFraudConfigDTO, error) {
	var out CampaignFraudConfigDTO

	err := pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapNotFound(err, ErrCampaignNotFound)
		}

		pass := uint8(locked.FraudThresholdPass)
		suspect := uint8(locked.FraudThresholdSuspect)
		ivt := uint8(locked.FraudThresholdIvt)
		block := uint8(locked.FraudThresholdBlock)
		silentReject := locked.SilentRejectEnabled
		flags := locked.BehaviorFlags

		if upd.Preset != nil {
			presetPass, presetSuspect, presetIVT, presetBlock, err := s.resolveFraudPresetThresholds(ctx, *upd.Preset)
			if err != nil {
				return err
			}
			pass = presetPass
			suspect = presetSuspect
			ivt = presetIVT
			block = presetBlock
		}

		if upd.FraudThresholdPass != nil {
			pass = *upd.FraudThresholdPass
		}
		if upd.FraudThresholdSuspect != nil {
			suspect = *upd.FraudThresholdSuspect
		}
		if upd.FraudThresholdIVT != nil {
			ivt = *upd.FraudThresholdIVT
		}
		if upd.FraudThresholdBlock != nil {
			block = *upd.FraudThresholdBlock
		}
		if upd.SilentRejectEnabled != nil {
			silentReject = *upd.SilentRejectEnabled
		}
		if upd.BehaviorFlags != nil {
			flags = int32(*upd.BehaviorFlags)
		}

		if err := validateFraudThresholds(pass, suspect, ivt, block); err != nil {
			return err
		}

		enhancedDefensePreset := upd.Preset != nil && domain.IsEnhancedDefenseFraudPreset(*upd.Preset)
		if enhancedDefensePreset {
			if err := applyEnhancedDefensePreset(ctx, tx, campaignID); err != nil {
				return err
			}
			silentReject = true
		}
		socialInAppPreset := upd.Preset != nil && domain.IsSocialInAppFraudPreset(*upd.Preset)
		if socialInAppPreset {
			if err := applySocialInAppPreset(ctx, tx, campaignID); err != nil {
				return err
			}
		}

		updated, err := q.UpdateCampaignFraudConfig(ctx, db.UpdateCampaignFraudConfigParams{
			ID:                    domain.ToUUID(campaignID),
			FraudThresholdPass:    int16(pass),
			FraudThresholdSuspect: int16(suspect),
			FraudThresholdIvt:     int16(ivt),
			FraudThresholdBlock:   int16(block),
			SilentRejectEnabled:   silentReject,
			BehaviorFlags:         flags,
		})
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "UPDATE_CAMPAIGN_FRAUD", "campaign", &campaignID, auditCampaignFraudChange{
			FraudThresholdPass:    pass,
			FraudThresholdSuspect: suspect,
			FraudThresholdIVT:     ivt,
			FraudThresholdBlock:   block,
			SilentRejectEnabled:   silentReject,
			BehaviorFlags:         flags,
		}, nil)

		payload, err := coldpath.MarshalOutbox(campaignIDPayload{CampaignID: campaignID.String()})
		if err != nil {
			return fmt.Errorf("marshal update campaign fraud outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_CAMPAIGN_FRAUD",
			Payload:   payload,
		})
		if err != nil {
			return err
		}

		out = CampaignFraudConfigDTO{
			CampaignID:            campaignID.String(),
			FraudThresholdPass:    uint8(updated.FraudThresholdPass),
			FraudThresholdSuspect: uint8(updated.FraudThresholdSuspect),
			FraudThresholdIVT:     uint8(updated.FraudThresholdIvt),
			FraudThresholdBlock:   uint8(updated.FraudThresholdBlock),
			SilentRejectEnabled:   updated.SilentRejectEnabled,
			BehaviorFlags:         uint32(updated.BehaviorFlags),
		}
		return nil
	})
	if err != nil {
		return CampaignFraudConfigDTO{}, err
	}
	return out, nil
}

func ResolveFraudThresholds(camp *domain.Campaign) (pass, suspect, ivt, block uint8) {
	if camp == nil {
		return domain.DefaultFraudThresholdPass, domain.DefaultFraudThresholdSuspect,
			domain.DefaultFraudThresholdIVT, domain.DefaultFraudThresholdBlock
	}
	return camp.FraudThresholdPass, camp.FraudThresholdSuspect, camp.FraudThresholdIVT, camp.FraudThresholdBlock
}

type FraudScoringOverrideRequest struct {
	CampaignID *string `json:"campaign_id,omitempty"`
	IP         *string `json:"ip,omitempty"`
}

func normalizeFraudLabelLimit(limit int) int {
	if limit <= 0 {
		return fraudManualLabelsDefaultLimit
	}
	if limit > fraudManualLabelsMaxLimit {
		return fraudManualLabelsMaxLimit
	}
	return limit
}

func validateMLIPHash(ipHash string) error {
	if ipHash == "" {
		return errValidation("ip_hash required")
	}
	if len(ipHash) != 32 {
		return errValidation("ip_hash must be 32 hex characters")
	}
	for _, c := range ipHash {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return errValidation("ip_hash must be 32 hex characters")
	}
	return nil
}

func (s *Service) ListMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, limit int) ([]MLManualLabelDTO, error) {
	if customerID == uuid.Nil {
		return nil, errValidation("customer_id is required")
	}
	if s == nil || s.GetPool() == nil {
		return nil, fmt.Errorf("postgres pool not configured")
	}
	limit = normalizeFraudLabelLimit(limit)
	rows, err := s.GetPool().Query(ctx, `
		SELECT ip_hash, label, reason, source, created_at
		FROM ml_manual_labels
		WHERE customer_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, domain.ToUUID(customerID), limit)
	if err != nil {
		return nil, fmt.Errorf("query ml_manual_labels: %w", err)
	}
	defer rows.Close()

	out := make([]MLManualLabelDTO, 0, limit)
	for rows.Next() {
		var row MLManualLabelDTO
		var createdAt time.Time
		if err := rows.Scan(&row.IPHash, &row.Label, &row.Reason, &row.Source, &createdAt); err != nil {
			return nil, err
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Service) UpsertMLManualLabelForCustomer(ctx context.Context, customerID uuid.UUID, ipHash string, label int, reason string) error {
	if customerID == uuid.Nil {
		return errValidation("customer_id is required")
	}
	if err := validateMLIPHash(ipHash); err != nil {
		return err
	}
	if label != 0 && label != 1 {
		return errValidation("label must be 0 or 1")
	}
	if s == nil || s.GetPool() == nil {
		return fmt.Errorf("postgres pool not configured")
	}
	_, err := s.GetPool().Exec(ctx, `
		INSERT INTO ml_manual_labels (ip_hash, label, reason, source, customer_id, created_at)
		VALUES ($1, $2, $3, 'admin_ui', $4, NOW())
		ON CONFLICT (ip_hash) DO UPDATE SET
			label = EXCLUDED.label,
			reason = EXCLUDED.reason,
			source = EXCLUDED.source,
			customer_id = EXCLUDED.customer_id,
			created_at = NOW()`,
		ipHash, label, reason, domain.ToUUID(customerID))
	return err
}

type MLManualLabelInput struct {
	IPHash string
	Label  int
	Reason string
}

func (s *Service) BulkUpsertMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, rows []MLManualLabelInput) (int, error) {
	if customerID == uuid.Nil {
		return 0, errValidation("customer_id is required")
	}
	if len(rows) == 0 {
		return 0, errValidation("rows required")
	}
	if len(rows) > fraudManualLabelsBulkMax {
		return 0, errValidation(fmt.Sprintf("max %d rows per bulk request", fraudManualLabelsBulkMax))
	}
	if s == nil || s.GetPool() == nil {
		return 0, fmt.Errorf("postgres pool not configured")
	}

	var inserted int
	err := pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		batch := &pgx.Batch{}
		for i, row := range rows {
			if err := validateMLIPHash(row.IPHash); err != nil {
				return fmt.Errorf("row %d: %w", i+1, err)
			}
			if row.Label != 0 && row.Label != 1 {
				return fmt.Errorf("row %d: label must be 0 or 1", i+1)
			}
			batch.Queue(`
				INSERT INTO ml_manual_labels (ip_hash, label, reason, source, customer_id, created_at)
				VALUES ($1, $2, $3, 'admin_ui', $4, NOW())
				ON CONFLICT (ip_hash) DO UPDATE SET
					label = EXCLUDED.label,
					reason = EXCLUDED.reason,
					source = EXCLUDED.source,
					customer_id = EXCLUDED.customer_id,
					created_at = NOW()`,
				row.IPHash, row.Label, row.Reason, domain.ToUUID(customerID))
		}
		br := tx.SendBatch(ctx, batch)
		for range rows {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return err
			}
			inserted++
		}
		return br.Close()
	})
	if err != nil {
		return 0, err
	}
	return inserted, nil
}

func (s *Service) ApplyFraudScoringOverride(ctx context.Context, req FraudScoringOverrideRequest) error {
	hasCampaign := req.CampaignID != nil && strings.TrimSpace(*req.CampaignID) != ""
	hasIP := req.IP != nil && strings.TrimSpace(*req.IP) != ""
	if !hasCampaign && !hasIP {
		return errValidation("at least one of campaign_id or ip is required")
	}
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}

		if req.CampaignID != nil && *req.CampaignID != "" {
			campUUID, err := uuid.Parse(*req.CampaignID)
			if err != nil {
				return errValidation("invalid campaign_id format")
			}

			s.AuditLog(ctx, q, uid, "FRAUD_CLEAR_BOOST", "campaign", &campUUID, map[string]string{"campaign_id": *req.CampaignID}, nil)

			payload, err := coldpath.MarshalOutbox(FraudThreatPayload{
				Action:     "boost",
				CampaignID: *req.CampaignID,
				Boost:      0,
				TTLSeconds: 0,
			})
			if err != nil {
				return err
			}
			_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
				EventType: "ML_SCORE_BOOST",
				Payload:   payload,
			})
			if err != nil {
				return err
			}
		}

		if req.IP != nil && *req.IP != "" {
			err := q.DeleteBlacklistIP(ctx, *req.IP)
			if err != nil {
				return err
			}

			s.AuditLog(ctx, q, uid, "FRAUD_REMOVE_FALSE_POSITIVE", "system", nil, map[string]string{"ip": *req.IP}, nil)

			payload, err := coldpath.MarshalOutbox(BlacklistPayload{Action: "remove", IP: *req.IP, Reason: "fraud"})
			if err != nil {
				return fmt.Errorf("marshal blacklist outbox payload: %w", err)
			}
			_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
				EventType: "UPDATE_BLACKLIST",
				Payload:   payload,
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Service) CheckAndHandleStaleEpochs(ctx context.Context) error {
	if len(s.redisShards) == 0 {
		return nil
	}

	now := time.Now().Unix()
	var maxAppliedAt int64
	var foundStale bool

	for _, redisClient := range s.redisShards {
		if redisClient == nil {
			continue
		}
		val, err := redisClient.Get(ctx, "ml:model:applied_at").Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			return fmt.Errorf("query ml:model:applied_at: %w", err)
		}
		appliedAt, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}
		if appliedAt > maxAppliedAt {
			maxAppliedAt = appliedAt
		}
	}

	if maxAppliedAt > 0 && now-maxAppliedAt > 600 {
		foundStale = true
	}

	if !foundStale {
		return nil
	}

	var currentPctStr string
	err := s.pool.QueryRow(ctx, "SELECT value FROM system_settings WHERE key = 'fraud_rl_suspect_pct'").Scan(&currentPctStr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			currentPctStr = "50"
		} else {
			return err
		}
	}

	currentPct, _ := strconv.Atoi(currentPctStr)
	newPct := currentPct / 2
	if newPct < 10 {
		newPct = 10
	}

	if newPct == currentPct {
		return nil
	}

	if err := s.UpdateSettings(ctx, map[string]string{
		"fraud_rl_suspect_pct": strconv.Itoa(newPct),
	}); err != nil {
		return fmt.Errorf("tighten suspect rate limit: %w", err)
	}
	return nil
}

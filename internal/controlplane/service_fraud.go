package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type CampaignFraudConfigDTO struct {
	CampaignID            string `json:"campaign_id"`
	FraudThresholdPass    uint8  `json:"fraud_threshold_pass"`
	FraudThresholdSuspect uint8  `json:"fraud_threshold_suspect"`
	FraudThresholdIVT     uint8  `json:"fraud_threshold_ivt"`
	FraudThresholdBlock   uint8  `json:"fraud_threshold_block"`
	GhostIVTEnabled       bool   `json:"ghost_ivt_enabled"`
	BehaviorFlags         uint32 `json:"behavior_flags"`
}

type CampaignFraudConfigUpdate struct {
	FraudThresholdPass    *uint8  `json:"fraud_threshold_pass,omitempty"`
	FraudThresholdSuspect *uint8  `json:"fraud_threshold_suspect,omitempty"`
	FraudThresholdIVT     *uint8  `json:"fraud_threshold_ivt,omitempty"`
	FraudThresholdBlock   *uint8  `json:"fraud_threshold_block,omitempty"`
	GhostIVTEnabled       *bool   `json:"ghost_ivt_enabled,omitempty"`
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
		GhostIVTEnabled:       row.GhostIvtEnabled,
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
		ghost := locked.GhostIvtEnabled
		flags := locked.BehaviorFlags

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
		if upd.GhostIVTEnabled != nil {
			ghost = *upd.GhostIVTEnabled
		}
		if upd.BehaviorFlags != nil {
			flags = int32(*upd.BehaviorFlags)
		}

		if err := validateFraudThresholds(pass, suspect, ivt, block); err != nil {
			return err
		}

		updated, err := q.UpdateCampaignFraudConfig(ctx, db.UpdateCampaignFraudConfigParams{
			ID:                    domain.ToUUID(campaignID),
			FraudThresholdPass:    int16(pass),
			FraudThresholdSuspect: int16(suspect),
			FraudThresholdIvt:     int16(ivt),
			FraudThresholdBlock:   int16(block),
			GhostIvtEnabled:       ghost,
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
			GhostIVTEnabled:       ghost,
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
			GhostIVTEnabled:       updated.GhostIvtEnabled,
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

func (s *Service) ApplyFraudScoringOverride(ctx context.Context, req FraudScoringOverrideRequest) error {
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
	if len(s.rdbs) == 0 {
		return nil
	}

	now := time.Now().Unix()
	var maxAppliedAt int64
	var foundStale bool

	for _, rdb := range s.rdbs {
		if rdb == nil {
			continue
		}
		val, err := rdb.Get(ctx, "ml:model:applied_at").Result()
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

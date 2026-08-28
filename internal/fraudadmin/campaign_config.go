package fraudadmin

import (
	"context"
	"fmt"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func validateFraudThresholds(pass, suspect, ivt, block uint8) error {
	if err := ValidateThresholds(pass, suspect, ivt, block); err != nil {
		return err
	}
	return nil
}

func GetCampaignFraudConfig(ctx context.Context, host CampaignConfigHost, campaignID uuid.UUID) (campaign.CampaignFraudConfigDTO, error) {
	if host == nil || host.ConfigPool() == nil {
		return campaign.CampaignFraudConfigDTO{}, fmt.Errorf("postgres pool not configured")
	}
	row, err := db.New(host.ConfigPool()).GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return campaign.CampaignFraudConfigDTO{}, mapCampaignNotFound(err)
	}
	return campaign.CampaignFraudConfigDTO{
		CampaignID:               campaignID.String(),
		FraudThresholdPass:       uint8(row.FraudThresholdPass),
		FraudThresholdSuspect:    uint8(row.FraudThresholdSuspect),
		FraudThresholdIVT:        uint8(row.FraudThresholdIvt),
		FraudThresholdBlock:      uint8(row.FraudThresholdBlock),
		SilentRejectEnabled:      row.SilentRejectEnabled,
		BehaviorFlags:            uint32(row.BehaviorFlags),
		CanvasRetestEnabled:      row.CanvasRetestEnabled,
		CgnatIPPolicyEnabled:     row.CgnatIpPolicyEnabled,
		AcceptLangGeoEnabled:     row.AcceptLangGeoEnabled,
		JSONSerializationEnabled: row.JsonSerializationEnabled,
		ConversionRejectRules:    domain.ParseConversionRejectRulesJSON(row.ConversionRejectRules),
	}, nil
}

func UpdateCampaignFraudConfig(ctx context.Context, host CampaignConfigHost, campaignID uuid.UUID, upd campaign.PatchCampaignFraudRequest) (campaign.CampaignFraudConfigDTO, error) {
	var out campaign.CampaignFraudConfigDTO
	if host == nil || host.ConfigPool() == nil {
		return out, fmt.Errorf("postgres pool not configured")
	}

	err := pgx.BeginFunc(ctx, host.ConfigPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapCampaignNotFound(err)
		}

		pass := uint8(locked.FraudThresholdPass)
		suspect := uint8(locked.FraudThresholdSuspect)
		ivt := uint8(locked.FraudThresholdIvt)
		block := uint8(locked.FraudThresholdBlock)
		silentReject := locked.SilentRejectEnabled
		flags := locked.BehaviorFlags
		canvasRetest := locked.CanvasRetestEnabled
		cgnatPolicy := locked.CgnatIpPolicyEnabled
		acceptLangGeo := locked.AcceptLangGeoEnabled
		jsonSerialization := locked.JsonSerializationEnabled
		conversionRejectRules := domain.ParseConversionRejectRulesJSON(locked.ConversionRejectRules)

		if upd.Preset != nil {
			presetPass, presetSuspect, presetIVT, presetBlock, err := host.ConfigResolvePresetThresholds(ctx, *upd.Preset)
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
		if upd.CanvasRetestEnabled != nil {
			canvasRetest = *upd.CanvasRetestEnabled
		}
		if upd.CgnatIPPolicyEnabled != nil {
			cgnatPolicy = *upd.CgnatIPPolicyEnabled
		}
		if upd.AcceptLangGeoEnabled != nil {
			acceptLangGeo = *upd.AcceptLangGeoEnabled
		}
		if upd.JSONSerializationEnabled != nil {
			jsonSerialization = *upd.JSONSerializationEnabled
		}
		if upd.ConversionRejectRules != nil {
			conversionRejectRules = domain.MergeConversionRejectRulesPatch(conversionRejectRules, *upd.ConversionRejectRules)
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
			acceptLangGeo = true
		}
		socialInAppPreset := upd.Preset != nil && domain.IsSocialInAppFraudPreset(*upd.Preset)
		if socialInAppPreset {
			if err := applySocialInAppPreset(ctx, tx, campaignID); err != nil {
				return err
			}
		}

		rulesBytes, err := domain.MarshalConversionRejectRules(conversionRejectRules)
		if err != nil {
			return err
		}

		updated, err := q.UpdateCampaignFraudConfig(ctx, db.UpdateCampaignFraudConfigParams{
			ID:                       domain.ToUUID(campaignID),
			FraudThresholdPass:       int16(pass),
			FraudThresholdSuspect:    int16(suspect),
			FraudThresholdIvt:        int16(ivt),
			FraudThresholdBlock:      int16(block),
			SilentRejectEnabled:      silentReject,
			BehaviorFlags:            flags,
			CanvasRetestEnabled:      canvasRetest,
			CgnatIpPolicyEnabled:     cgnatPolicy,
			AcceptLangGeoEnabled:     acceptLangGeo,
			JsonSerializationEnabled: jsonSerialization,
			ConversionRejectRules:    rulesBytes,
		})
		if err != nil {
			return err
		}

		adminID := host.ConfigActorID(ctx)
		host.ConfigAuditUpdate(ctx, q, adminID, campaignID, CampaignFraudAuditChange{
			FraudThresholdPass:       pass,
			FraudThresholdSuspect:    suspect,
			FraudThresholdIVT:        ivt,
			FraudThresholdBlock:      block,
			SilentRejectEnabled:      silentReject,
			BehaviorFlags:            flags,
			CanvasRetestEnabled:      canvasRetest,
			CgnatIPPolicyEnabled:     cgnatPolicy,
			AcceptLangGeoEnabled:     acceptLangGeo,
			JSONSerializationEnabled: jsonSerialization,
		})

		if err := host.ConfigEnqueueUpdateCampaignFraud(ctx, q, campaignID); err != nil {
			return err
		}

		out = campaign.CampaignFraudConfigDTO{
			CampaignID:               campaignID.String(),
			FraudThresholdPass:       uint8(updated.FraudThresholdPass),
			FraudThresholdSuspect:    uint8(updated.FraudThresholdSuspect),
			FraudThresholdIVT:        uint8(updated.FraudThresholdIvt),
			FraudThresholdBlock:      uint8(updated.FraudThresholdBlock),
			SilentRejectEnabled:      updated.SilentRejectEnabled,
			BehaviorFlags:            uint32(updated.BehaviorFlags),
			CanvasRetestEnabled:      updated.CanvasRetestEnabled,
			CgnatIPPolicyEnabled:     updated.CgnatIpPolicyEnabled,
			AcceptLangGeoEnabled:     updated.AcceptLangGeoEnabled,
			JSONSerializationEnabled: updated.JsonSerializationEnabled,
			ConversionRejectRules:    domain.ParseConversionRejectRulesJSON(updated.ConversionRejectRules),
		}
		return nil
	})
	if err != nil {
		return campaign.CampaignFraudConfigDTO{}, err
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

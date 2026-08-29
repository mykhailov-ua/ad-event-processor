package campaign

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ApplyClickPresetPatch(
	ctx context.Context,
	fx Effects,
	pool *pgxpool.Pool,
	campaignID uuid.UUID,
	templateID *string,
	params *map[string]string,
) error {
	if pool == nil || fx == nil {
		return errServiceUnavailable()
	}
	if templateID == nil && params == nil {
		return nil
	}
	camp, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := assertMediaBuyerCampaignAccess(ctx, camp); err != nil {
		return err
	}

	nextTemplate := FormatOptionalText(camp.TrafficTemplateID)
	if templateID != nil {
		nextTemplate = strings.TrimSpace(*templateID)
		if err := validateTrafficTemplateID(nextTemplate); err != nil {
			return err
		}
	}
	nextParams := ClickQueryParamsFromRaw(camp.ClickQueryParams)
	if params != nil {
		if err := validateClickQueryParams(*params); err != nil {
			return err
		}
		nextParams = normalizeClickQueryParams(*params)
	}

	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		return ApplyCampaignClickPresetTx(ctx, tx, campaignID, nextTemplate, nextParams)
	})
	if err != nil {
		return err
	}
	fx.PublishCampaignUpdate(ctx, campaignID.String())
	return nil
}

func ApplyCampaignClickPresetTx(ctx context.Context, tx pgx.Tx, campaignID uuid.UUID, templateID string, params map[string]string) error {
	templateID = strings.TrimSpace(templateID)
	params = normalizeClickQueryParams(params)
	if templateID == "" && len(params) == 0 {
		return nil
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	if len(params) == 0 {
		raw = []byte("{}")
	}
	_, err = tx.Exec(ctx, `
UPDATE campaigns
SET traffic_template_id = $2,
    click_query_params = $3::jsonb,
    updated_at = NOW()
WHERE id = $1`, campaignID, nullString(templateID), raw)
	return err
}

func nullString(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

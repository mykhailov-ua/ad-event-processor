package campaign

import (
	"context"
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

package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	maxClickQueryParamKeys     = 40
	maxClickQueryParamValueLen = 512
	maxTrafficTemplateIDLen    = 64
)

var allowedClickQueryKeys = func() map[string]bool {
	keys := map[string]bool{
		"ad_campaign_id": true,
		"fbclid":         true,
		"gclid":          true,
		"ttclid":         true,
	}
	for i := 1; i <= 30; i++ {
		keys[fmt.Sprintf("sub%d", i)] = true
	}
	return keys
}()

func validateTrafficTemplateID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if len(id) > maxTrafficTemplateIDLen {
		return fmt.Errorf("traffic_template_id too long")
	}
	return nil
}

func validateClickQueryParams(params map[string]string) error {
	if len(params) > maxClickQueryParamKeys {
		return fmt.Errorf("click_query_params: too many keys")
	}
	for key, value := range params {
		if !allowedClickQueryKeys[key] {
			return fmt.Errorf("click_query_params: invalid key %q", key)
		}
		if len(value) > maxClickQueryParamValueLen {
			return fmt.Errorf("click_query_params: value too long for %q", key)
		}
	}
	return nil
}

func normalizeClickQueryParams(params map[string]string) map[string]string {
	if len(params) == 0 {
		return nil
	}
	out := make(map[string]string, len(params))
	for key, value := range params {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func applyCampaignClickPresetTx(ctx context.Context, tx pgx.Tx, campaignID uuid.UUID, templateID string, params map[string]string) error {
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

func (s *Service) applyCampaignClickPresetPatch(
	ctx context.Context,
	campaignID uuid.UUID,
	templateID *string,
	params *map[string]string,
) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("service unavailable")
	}
	if templateID == nil && params == nil {
		return nil
	}
	camp, err := s.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := assertMediaBuyerCampaignAccess(ctx, camp); err != nil {
		return err
	}

	nextTemplate := formatOptionalText(camp.TrafficTemplateID)
	if templateID != nil {
		nextTemplate = strings.TrimSpace(*templateID)
		if err := validateTrafficTemplateID(nextTemplate); err != nil {
			return err
		}
	}
	nextParams := clickQueryParamsFromRaw(camp.ClickQueryParams)
	if params != nil {
		nextParams = normalizeClickQueryParams(*params)
		if err := validateClickQueryParams(*params); err != nil {
			return err
		}
	}

	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		return applyCampaignClickPresetTx(ctx, tx, campaignID, nextTemplate, nextParams)
	})
	if err != nil {
		return err
	}
	_ = s.publishCampaignUpdate(ctx, campaignID.String())
	return nil
}

func nullString(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func clickQueryParamsFromRaw(raw []byte) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil || len(out) == 0 {
		return nil
	}
	return normalizeClickQueryParams(out)
}

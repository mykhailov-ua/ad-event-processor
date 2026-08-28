package campaign

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

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

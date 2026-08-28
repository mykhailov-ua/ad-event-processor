package costsync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	costSourceAPIToken  = "api_token"
	costSourceAPISpread = "api_spread"
)

type ClickCostAttributor interface {
	AttributeLines(ctx context.Context, runID int64, syncRunUUID uuid.UUID, mapping TokenMapping, lines []CostLine, usdMicro []int64, day time.Time) error
}

type noopAttributor struct{}

func (a noopAttributor) AttributeLines(context.Context, int64, uuid.UUID, TokenMapping, []CostLine, []int64, time.Time) error {
	return nil
}

func NewClickCostAttributor(pool *pgxpool.Pool, conn driver.Conn) ClickCostAttributor {
	if conn == nil {
		return noopAttributor{}
	}
	return &clickCostAttributor{pool: pool, conn: conn}
}

type clickCostAttributor struct {
	pool *pgxpool.Pool
	conn driver.Conn
}

func (a *clickCostAttributor) AttributeLines(ctx context.Context, runID int64, syncRunUUID uuid.UUID, mapping TokenMapping, lines []CostLine, usdMicro []int64, day time.Time) error {
	if a == nil || a.conn == nil || len(lines) == 0 {
		return nil
	}
	if mapping.AttributionMode == AttributionModeSpread {
		return a.attributeSpread(ctx, runID, mapping, lines, usdMicro, day)
	}
	return a.attributeToken(ctx, runID, mapping, lines, usdMicro, day)
}

func (a *clickCostAttributor) attributeToken(ctx context.Context, runID int64, mapping TokenMapping, lines []CostLine, usdMicro []int64, day time.Time) error {
	dayStart, dayEnd := dayBounds(day)
	whereField, err := clickMatchColumn(mapping.PlacementField)
	if err != nil {
		return err
	}

	for i, line := range lines {
		if line.LineType != LineTypeSpend {
			continue
		}
		token := strings.TrimSpace(line.PlacementID)
		if token == "" {
			continue
		}
		amount := line.AmountMicro
		if i < len(usdMicro) {
			amount = usdMicro[i]
		}
		if amount <= 0 {
			continue
		}
		if a.pool != nil {
			tag, err := a.pool.Exec(ctx, `
				INSERT INTO cost_sync_attribution_applied (sync_run_id, campaign_id, placement_id)
				VALUES ($1, $2, $3)
				ON CONFLICT DO NOTHING`, runID, line.CampaignID, token)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				continue
			}
		}
		query := fmt.Sprintf(`
			ALTER TABLE ad_event_processor.clicks
			UPDATE attributed_cost_micro = ?, cost_source = ?
			WHERE campaign_id = ?
			  AND %s = ?
			  AND created_at >= ?
			  AND created_at < ?
			  AND (cost_source = '' OR cost_source = 'ingress_macro')
			SETTINGS mutations_sync = 1`, whereField)
		if err := a.conn.Exec(ctx, query, amount, costSourceAPIToken, line.CampaignID, token, dayStart, dayEnd); err != nil {
			return err
		}
	}
	return nil
}

func (a *clickCostAttributor) attributeSpread(ctx context.Context, runID int64, mapping TokenMapping, lines []CostLine, usdMicro []int64, day time.Time) error {
	_ = mapping
	byCampaign := make(map[uuid.UUID]int64)
	for i, line := range lines {
		if line.LineType != LineTypeSpend {
			continue
		}
		amount := line.AmountMicro
		if i < len(usdMicro) {
			amount = usdMicro[i]
		}
		if amount <= 0 {
			continue
		}
		byCampaign[line.CampaignID] += amount
	}
	dayStart, dayEnd := dayBounds(day)

	for campaignID, totalMicro := range byCampaign {
		spreadKey := fmt.Sprintf("_spread_%s", campaignID)
		if a.pool != nil {
			tag, err := a.pool.Exec(ctx, `
				INSERT INTO cost_sync_attribution_applied (sync_run_id, campaign_id, placement_id)
				VALUES ($1, $2, $3)
				ON CONFLICT DO NOTHING`, runID, campaignID, spreadKey)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				continue
			}
		}
		var clickCount uint64
		if err := a.conn.QueryRow(ctx, `
			SELECT count()
			FROM ad_event_processor.clicks
			WHERE campaign_id = ?
			  AND created_at >= ?
			  AND created_at < ?
			  AND (cost_source = '' OR cost_source = 'ingress_macro')`,
			campaignID, dayStart, dayEnd).Scan(&clickCount); err != nil {
			return err
		}
		if clickCount == 0 {
			continue
		}
		perClick := totalMicro / int64(clickCount)
		if perClick <= 0 {
			continue
		}
		if err := a.conn.Exec(ctx, `
			ALTER TABLE ad_event_processor.clicks
			UPDATE attributed_cost_micro = ?, cost_source = ?
			WHERE campaign_id = ?
			  AND created_at >= ?
			  AND created_at < ?
			  AND (cost_source = '' OR cost_source = 'ingress_macro')
			SETTINGS mutations_sync = 1`,
			perClick, costSourceAPISpread, campaignID, dayStart, dayEnd); err != nil {
			return err
		}
	}
	return nil
}

func dayBounds(day time.Time) (time.Time, time.Time) {
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	return dayStart, dayStart.AddDate(0, 0, 1)
}

func clickMatchColumn(placementField string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(placementField)) {
	case "", "placement_id":
		return "placement_id", nil
	case "sub1", "sub2":
		return placementField, nil
	default:
		return "", fmt.Errorf("unsupported placement_field %q", placementField)
	}
}

package dashboardadmin

import (
	"context"
	"fmt"
	"sort"

	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const buyerBreakdownTopN = 6

type campaignBreakdownInput struct {
	id           string
	name         string
	clicks       int64
	uniqueClicks int64
	conversions  int64
	costMicro    int64
	revenueMicro int64
}

func buildCampaignBreakdownTable(inputs []campaignBreakdownInput) reports.DashboardBreakdownTableDTO {
	rows := make([]reports.DashboardBreakdownRowDTO, 0, len(inputs))
	for _, input := range inputs {
		if input.clicks == 0 && input.conversions == 0 && input.costMicro == 0 && input.revenueMicro == 0 {
			continue
		}
		profitMicro := input.revenueMicro - input.costMicro
		row := reports.DashboardBreakdownRowDTO{
			ID:           input.id,
			Name:         input.name,
			Clicks:       input.clicks,
			UniqueClicks: input.uniqueClicks,
			Conversions:  input.conversions,
			CostMicro:    input.costMicro,
			RevenueMicro: input.revenueMicro,
			ProfitMicro:  profitMicro,
		}
		reports.EnrichBreakdownEconomics(&row)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Clicks == rows[j].Clicks {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].Clicks > rows[j].Clicks
	})
	return reports.CapBreakdownTable(rows, buyerBreakdownTopN)
}

func attachFlowEntityNames(
	ctx context.Context,
	pool *pgxpool.Pool,
	table *reports.DashboardBreakdownTableDTO,
	entityTable string,
) error {
	if table == nil || len(table.Rows) == 0 || pool == nil {
		return nil
	}
	if entityTable != "landers" && entityTable != "offers" {
		return fmt.Errorf("invalid flow entity table %q", entityTable)
	}
	ids := make([]uuid.UUID, 0, len(table.Rows))
	for _, row := range table.Rows {
		raw := row.ID
		if raw == "" {
			raw = row.Name
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	names, err := lookupFlowEntityNamesPG(ctx, pool, entityTable, ids)
	if err != nil {
		return err
	}
	for i := range table.Rows {
		if name, ok := names[table.Rows[i].ID]; ok && name != "" {
			table.Rows[i].Name = name
			continue
		}
		if name, ok := names[table.Rows[i].Name]; ok && name != "" {
			table.Rows[i].ID = table.Rows[i].Name
			table.Rows[i].Name = name
		}
	}
	return nil
}

func lookupFlowEntityNamesPG(
	ctx context.Context,
	pool *pgxpool.Pool,
	entityTable string,
	ids []uuid.UUID,
) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	if pool == nil || len(ids) == 0 {
		return out, nil
	}
	query := fmt.Sprintf(`SELECT id::text, name FROM %s WHERE id = ANY($1)`, entityTable)
	rows, err := pool.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = name
	}
	return out, rows.Err()
}

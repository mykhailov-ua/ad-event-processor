import type { CampaignForecast } from '@/api/types';
import { DashboardKpiStrip, type DashboardKpiTile } from '@/domains/dashboards/dashboard_kpi_strip';
import { formatDashboardUsdFromMicro } from '@/domains/dashboards/dashboard_format';
import { Badge } from '@/components/ui/badge';
import { ReportMapTable } from '@/shell/report_map_table';

const SPEND_CURVE_COLUMNS = ['hour', 'spend_micro', 'impressions'] as const;

export function CampaignForecastView({ forecast }: { forecast: CampaignForecast }) {
  const kpiTiles: DashboardKpiTile[] = [
    { id: 'impressions_p50', label: 'Impressions (p50)', value: String(forecast.impressions_p50) },
    { id: 'impressions_p90', label: 'Impressions (p90)', value: String(forecast.impressions_p90) },
  ];

  const spendRows = (forecast.spend_curve ?? []).map((point) => ({
    hour: point.hour ?? '',
    spend_micro:
      point.spend_micro != null ? formatDashboardUsdFromMicro(point.spend_micro) : '',
    impressions: point.impressions ?? '',
  }));

  const advisory = forecast.advisory;

  return (
    <div className="grid gap-4">
      <div className="grid gap-4">
        <DashboardKpiStrip tiles={kpiTiles} />
        {forecast.low_confidence ? (
          <Badge className="w-fit" variant="secondary">
            Low confidence
          </Badge>
        ) : null}
      </div>

      {advisory?.message ? (
        <div className="ui-surface-raised grid gap-1 p-4 text-sm">
          {advisory.code ? (
            <p className="font-medium text-muted-foreground">{advisory.code}</p>
          ) : null}
          <p>{advisory.message}</p>
          {advisory.suggested_pacing ? (
            <p className="text-muted-foreground">
              Suggested pacing: {advisory.suggested_pacing}
            </p>
          ) : null}
        </div>
      ) : null}

      {spendRows.length > 0 ? (
        <ReportMapTable
          caption="Spend curve"
          columns={SPEND_CURVE_COLUMNS}
          rowKeyPrefix="spend-curve"
          rows={spendRows}
        />
      ) : null}
    </div>
  );
}

import type { PublisherDashboard } from '@/api/types';
import { DashboardKpiStrip, type DashboardKpiTile } from '@/domains/dashboards/dashboard_kpi_strip';
import { formatDashboardUsdFromMicro } from '@/domains/dashboards/dashboard_format';
import { displayTimestamp } from '@/lib/display';
import { ReportMapTable } from '@/shell/report_map_table';

function buildPublisherKpiTiles(kpis: PublisherDashboard['kpis']): DashboardKpiTile[] {
  if (!kpis) {
    return [];
  }
  const tiles: DashboardKpiTile[] = [];
  if (kpis.impressions != null) {
    tiles.push({ id: 'impressions', label: 'Impressions', value: String(kpis.impressions) });
  }
  if (kpis.fill_rate != null) {
    tiles.push({
      id: 'fill_rate',
      label: 'Fill rate',
      value: `${(kpis.fill_rate * 100).toFixed(2)}%`,
    });
  }
  if (kpis.ecpm_micro != null) {
    tiles.push({
      id: 'ecpm',
      label: 'eCPM',
      value: formatDashboardUsdFromMicro(kpis.ecpm_micro) || '0.00',
    });
  }
  if (kpis.ivt_rate != null) {
    tiles.push({
      id: 'ivt_rate',
      label: 'IVT rate',
      value: `${(kpis.ivt_rate * 100).toFixed(2)}%`,
    });
  }
  return tiles;
}

const PLACEMENT_COLUMNS = [
  'placement_id',
  'impressions',
  'clicks',
  'fill_rate',
  'revenue_micro',
  'ecpm_micro',
] as const;

export function PublisherDashboardView({ dashboard }: { dashboard: PublisherDashboard }) {
  const kpiTiles = buildPublisherKpiTiles(dashboard.kpis);
  const placements = dashboard.placements ?? [];
  const placementRows = placements.map((row) => ({
    placement_id: row.placement_id ?? '',
    impressions: row.impressions ?? '',
    clicks: row.clicks ?? '',
    fill_rate: row.fill_rate != null ? `${(row.fill_rate * 100).toFixed(2)}%` : '',
    revenue_micro: row.revenue_micro != null ? formatDashboardUsdFromMicro(row.revenue_micro) : '',
    ecpm_micro: row.ecpm_micro != null ? formatDashboardUsdFromMicro(row.ecpm_micro) : '',
  }));

  return (
    <div className="grid gap-4">
      {(dashboard.seller_id || dashboard.publisher_account_id || dashboard.from) && (
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-muted-foreground">
          {dashboard.seller_id ? <span>Seller: {dashboard.seller_id}</span> : null}
          {dashboard.publisher_account_id ? (
            <span>Account: {dashboard.publisher_account_id}</span>
          ) : null}
          {dashboard.from && dashboard.to ? (
            <span>
              {displayTimestamp(dashboard.from)} to {displayTimestamp(dashboard.to)}
            </span>
          ) : null}
        </div>
      )}

      {kpiTiles.length > 0 ? <DashboardKpiStrip tiles={kpiTiles} /> : null}

      {placementRows.length > 0 ? (
        <ReportMapTable
          caption="Placements"
          columns={PLACEMENT_COLUMNS}
          rowKeyPrefix="placement"
          rows={placementRows}
        />
      ) : null}
    </div>
  );
}

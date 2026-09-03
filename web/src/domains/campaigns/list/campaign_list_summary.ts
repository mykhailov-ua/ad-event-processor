import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import {
  emptyCampaignListTotals,
  formatTableCount,
  formatTableMoneyFromMicro,
  type CampaignListTotals,
} from '@/domains/campaigns/list/campaign_list_format';
import { resolveCampaignListRowMetrics } from '@/domains/campaigns/list/campaign_list_row_metrics';

export type CampaignListSummary = CampaignListTotals & {
  scope: 'page' | 'selection';
  rowCount: number;
  staleCount: number;
  marginBreachCount: number;
};

// Page or selection scope only; not the server-filtered list total.
export function computeCampaignListSummary(
  items: Campaign[],
  selectedIds: Set<string>,
  metricsById: Record<string, CampaignListMetrics>,
  marginsById: Record<string, CampaignMargin>,
): CampaignListSummary {
  const scoped =
    selectedIds.size > 0 ? items.filter((item) => selectedIds.has(item.id)) : items;
  const totals = emptyCampaignListTotals();
  let staleCount = 0;
  let marginBreachCount = 0;

  for (const campaign of scoped) {
    if (campaign.flow_id) {
      totals.flows += 1;
    }
    const metrics = metricsById[campaign.id];
    totals.clicks += metrics?.clicks ?? 0;
    totals.conversions += metrics?.conversions ?? 0;
    if (metrics?.stale) {
      staleCount += 1;
    }
    const margin = marginsById[campaign.id];
    if (margin?.margin_breach) {
      marginBreachCount += 1;
    }
    const row = resolveCampaignListRowMetrics(metrics, margin);
    totals.revenueMicro += row.revenueMicro;
    totals.costMicro += row.costMicro;
    totals.profitMicro += row.profitMicro;
  }

  return {
    ...totals,
    scope: selectedIds.size > 0 ? 'selection' : 'page',
    rowCount: scoped.length,
    staleCount,
    marginBreachCount,
  };
}

export function formatCampaignListSummaryLine(summary: CampaignListSummary): string {
  const clicks = formatTableCount(summary.clicks).text;
  const leads = formatTableCount(summary.conversions).text;
  const profit = formatTableMoneyFromMicro(summary.profitMicro).text;
  const scopeLabel = summary.scope === 'selection' ? `${summary.rowCount} selected` : 'page';
  return `${scopeLabel}: ${clicks} clicks, ${leads} leads, ${profit} profit`;
}

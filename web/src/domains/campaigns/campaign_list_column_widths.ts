import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import {
  campaignListConversionsLabel,
  campaignListCostLabel,
  campaignListClicksLabel,
  campaignListMarginCostLabel,
  campaignListProfitLabel,
  campaignListRevenueLabel,
  formatCampaignListCr,
  formatCampaignListRoi,
  sumCampaignListTotals,
} from '@/domains/campaigns/campaign_list_format';
import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX,
  type CampaignListColumnId,
  type CampaignListMiddleColumnId,
} from '@/domains/campaigns/campaign_list_columns';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/campaign_metrics_shared';
import { formatDashboardUsdFromMicro } from '@/domains/dashboards/dashboard_format';

const CELL_HORIZONTAL_PADDING_PX = 16;
const HEADER_EXTRA_PX = 30;
const CHAR_WIDTH_PX = 7.25;

function campaignDisplayId(id: string): string {
  let hash = 0;
  for (let index = 0; index < id.length; index += 1) {
    hash = (hash * 31 + id.charCodeAt(index)) >>> 0;
  }
  return String((hash % 9000) + 1000);
}

function estimateTextWidthPx(text: string): number {
  return Math.ceil(text.length * CHAR_WIDTH_PX);
}

function columnContentWidth(text: string, minWidth: number, header = false): number {
  const extra = header ? HEADER_EXTRA_PX : 0;
  return Math.max(minWidth, estimateTextWidthPx(text) + CELL_HORIZONTAL_PADDING_PX + extra);
}

export function campaignListMiddleCellText(
  columnId: CampaignListMiddleColumnId,
  campaign: Campaign,
  metrics: CampaignListMetrics | undefined,
  margin: CampaignMargin | undefined,
  customerName: string,
): string {
  const row = campaign as CampaignWithMoneyDisplay;

  switch (columnId) {
    case 'source':
      return campaign.traffic_template_id || 'Direct';
    case 'flows':
      return campaign.flow_id ? '1' : '0';
    case 'clicks':
      return campaignListClicksLabel(metrics) || '0';
    case 'conversions':
      return campaignListConversionsLabel(metrics) || '0';
    case 'cr':
      return formatCampaignListCr(metrics?.clicks, metrics?.conversions) || '0.00%';
    case 'revenue':
      return campaignListRevenueLabel(margin) || campaignListCostLabel(row) || '—';
    case 'cost':
      return campaignListMarginCostLabel(margin) || campaignListCostLabel(row) || '—';
    case 'profit':
      return campaignListProfitLabel(margin) || '—';
    case 'roi': {
      const roi = formatCampaignListRoi(margin?.operator_margin_micro, margin?.rtb_cost_micro);
      return roi || '—';
    }
    case 'group':
      return customerName;
    default:
      return '';
  }
}

function campaignListTotalsCellText(
  columnId: CampaignListColumnId,
  totals: ReturnType<typeof sumCampaignListTotals>,
  pageCount: number,
): string {
  switch (columnId) {
    case 'name':
      return 'Total';
    case 'flows':
      return String(totals.flows);
    case 'clicks':
      return String(totals.clicks);
    case 'conversions':
      return String(totals.conversions);
    case 'cr':
      return formatCampaignListCr(totals.clicks, totals.conversions) || '0.00%';
    case 'revenue':
      return formatDashboardUsdFromMicro(totals.revenueMicro) || '—';
    case 'cost':
      return formatDashboardUsdFromMicro(totals.costMicro) || '—';
    case 'profit':
      return formatDashboardUsdFromMicro(totals.profitMicro) || '—';
    case 'roi': {
      const roi = formatCampaignListRoi(totals.profitMicro, totals.costMicro);
      return roi || '—';
    }
    case 'group':
      return `${pageCount} on page`;
    default:
      return '';
  }
}

export function defaultCampaignListColumnWidths(
  columns: ReadonlyArray<CampaignListColumnId>,
): Record<CampaignListColumnId, number> {
  const widths = {} as Record<CampaignListColumnId, number>;
  for (const columnId of columns) {
    widths[columnId] = CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId];
  }
  return widths;
}

export function computeCampaignListColumnWidths({
  columns,
  items,
  metricsById,
  marginsById,
  customerNameById,
}: {
  columns: ReadonlyArray<CampaignListColumnId>;
  items: readonly Campaign[];
  metricsById: Readonly<Record<string, CampaignListMetrics>>;
  marginsById: Readonly<Record<string, CampaignMargin>>;
  customerNameById: Readonly<Record<string, string>>;
}): Record<CampaignListColumnId, number> {
  const widths = defaultCampaignListColumnWidths(columns);
  const totals = sumCampaignListTotals(
    items as CampaignWithMoneyDisplay[],
    metricsById,
    marginsById,
  );

  for (const columnId of columns) {
    const label = CAMPAIGN_LIST_COLUMN_LABELS[columnId];
    let maxWidth = columnContentWidth(label, widths[columnId], true);

    if (columnId === 'id') {
      for (const campaign of items) {
        maxWidth = Math.max(
          maxWidth,
          columnContentWidth(campaignDisplayId(campaign.id), widths[columnId]),
        );
      }
    } else if (columnId === 'name') {
      for (const campaign of items) {
        maxWidth = Math.max(maxWidth, columnContentWidth(campaign.name ?? '', widths[columnId]) + 14);
      }
    } else if (columnId !== 'select') {
      for (const campaign of items) {
        const text = campaignListMiddleCellText(
          columnId as CampaignListMiddleColumnId,
          campaign,
          metricsById[campaign.id],
          marginsById[campaign.id],
          customerNameById[campaign.customer_id] ?? campaign.customer_id,
        );
        maxWidth = Math.max(maxWidth, columnContentWidth(text, widths[columnId]));
      }
    }

    const totalsText = campaignListTotalsCellText(columnId, totals, items.length);
    if (totalsText) {
      maxWidth = Math.max(maxWidth, columnContentWidth(totalsText, widths[columnId]));
    }

    widths[columnId] = maxWidth;
  }

  return widths;
}

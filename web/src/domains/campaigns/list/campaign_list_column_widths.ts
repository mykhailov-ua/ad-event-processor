import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import {
  campaignListClicksLabel,
  campaignListCostLabel,
  campaignListMarginCostLabel,
  campaignListProfitLabel,
  campaignListRevenueLabel,
  formatCampaignListCr,
  formatCampaignListRoi,
  formatTableMoneyFromMicro,
  sumCampaignListTotals,
} from '@/domains/campaigns/list/campaign_list_format';
import {
  formatApproveRate,
  formatCpmUsd,
  formatLpCtr,
  formatRelativeRate,
  formatSourceCtr,
} from '@/domains/campaigns/list/campaign_list_funnel';
import { resolveCampaignListRowMetrics } from '@/domains/campaigns/list/campaign_list_row_metrics';
import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX,
  clampCampaignListColumnWidthPx,
  type CampaignListColumnId,
  type CampaignListMiddleColumnId,
} from '@/domains/campaigns/list/campaign_list_columns';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import { formatDashboardUsdFromMicro } from '@/domains/dashboards/dashboard_format';
import { campaignDisplayId } from '@/domains/campaigns/list/campaign_display_id';

const CELL_HORIZONTAL_PADDING_PX = 12;
const HEADER_EXTRA_PX = 24;
const CHAR_WIDTH_PX = 7;

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
  const { clicks, impressions, blocks, costMicro, revenueMicro, funnel } =
    resolveCampaignListRowMetrics(metrics, margin);

  switch (columnId) {
    case 'tags':
      return '';
    case 'clicks':
      return campaignListClicksLabel(metrics) || '0';
    case 'ctr':
      return formatSourceCtr(clicks, impressions);
    case 'lp_clicks':
      return String(funnel.lpClicks);
    case 'lp_views':
      return String(funnel.lpViews);
    case 'group':
      return customerName;
    case 'lp_ctr':
      return formatLpCtr(funnel.lpClicks, clicks);
    case 'cr':
      return formatCampaignListCr(clicks, funnel.approved) || '0.00%';
    case 'leads':
      return String(funnel.rawLeads);
    case 'approved':
      return String(funnel.approved);
    case 'hold_leads':
      return String(funnel.hold);
    case 'rejected_leads':
      return String(funnel.rejected);
    case 'approve_rate':
      return formatApproveRate(funnel.approved, funnel.rawLeads);
    case 'epc':
      return clicks > 0 ? formatTableMoneyFromMicro(Math.trunc(revenueMicro / clicks)).text : '0.00';
    case 'cpc':
      return clicks > 0 ? formatTableMoneyFromMicro(Math.trunc(costMicro / clicks)).text : '0.00';
    case 'cpa':
      return funnel.rawLeads > 0
        ? formatTableMoneyFromMicro(Math.trunc(costMicro / funnel.rawLeads)).text
        : '0.00';
    case 'ecpa':
      return funnel.approved > 0
        ? formatTableMoneyFromMicro(Math.trunc(costMicro / funnel.approved)).text
        : '0.00';
    case 'cpm':
      return formatCpmUsd(costMicro, impressions);
    case 'blocks':
      return String(blocks);
    case 'block_pct':
      return formatRelativeRate(blocks, clicks);
    case 'bots':
      return String(funnel.bots);
    case 'bot_pct':
      return formatRelativeRate(funnel.bots, clicks);
    case 'revenue':
      return campaignListRevenueLabel(margin) || campaignListCostLabel(row) || '0.00';
    case 'cost':
      return campaignListMarginCostLabel(margin) || campaignListCostLabel(row) || '0.00';
    case 'profit':
      return campaignListProfitLabel(margin) || '0.00';
    case 'roi': {
      const roi = formatCampaignListRoi(margin?.operator_margin_micro, margin?.rtb_cost_micro);
      return roi || '0%';
    }
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
    case 'clicks':
      return String(totals.clicks);
    case 'leads':
      return String(totals.conversions);
    case 'cr':
      return formatCampaignListCr(totals.clicks, totals.conversions) || '0.00%';
    case 'revenue':
      return formatDashboardUsdFromMicro(totals.revenueMicro) || '0.00';
    case 'cost':
      return formatDashboardUsdFromMicro(totals.costMicro) || '0.00';
    case 'profit':
      return formatDashboardUsdFromMicro(totals.profitMicro) || '0.00';
    case 'roi': {
      const roi = formatCampaignListRoi(totals.profitMicro, totals.costMicro);
      return roi || '0%';
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
          columnContentWidth(campaignDisplayId(campaign), widths[columnId]),
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

    widths[columnId] = clampCampaignListColumnWidthPx(columnId, maxWidth);
  }

  return widths;
}

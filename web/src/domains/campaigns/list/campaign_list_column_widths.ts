import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX,
  clampCampaignListColumnWidthPx,
  type CampaignListColumnId,
  type CampaignListMiddleColumnId,
} from '@/domains/campaigns/list/campaign_list_columns';
import { campaignDisplayId } from '@/domains/campaigns/list/campaign_display_id';
import { sumCampaignListTotals } from '@/domains/campaigns/list/campaign_list_format';
import { sumCampaignFunnelTotals } from '@/domains/campaigns/list/campaign_list_funnel';
import {
  buildCampaignRowVm,
  campaignListMiddleCellDisplayText,
} from '@/domains/campaigns/list/campaign_list_row_vm';
import { campaignListTotalsCellDisplayText } from '@/domains/campaigns/list/campaign_list_table_totals_display';
import type { CampaignListFilterTotalsView } from '@/domains/campaigns/list/campaign_list_filter_totals';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';

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
  customerNameById: Record<string, string>,
  ownerEmailById: Record<string, string> = {},
): string {
  const vm = buildCampaignRowVm(
    campaign,
    metrics,
    margin,
    customerNameById,
    ownerEmailById,
    false,
  );
  return campaignListMiddleCellDisplayText(columnId, vm);
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
  ownerEmailById = {},
  filterTotals,
}: {
  columns: ReadonlyArray<CampaignListColumnId>;
  items: readonly Campaign[];
  metricsById: Readonly<Record<string, CampaignListMetrics>>;
  marginsById: Readonly<Record<string, CampaignMargin>>;
  customerNameById: Readonly<Record<string, string>>;
  ownerEmailById?: Readonly<Record<string, string>>;
  filterTotals?: CampaignListFilterTotalsView;
}): Record<CampaignListColumnId, number> {
  const widths = defaultCampaignListColumnWidths(columns);
  const totals =
    filterTotals?.totals ??
    sumCampaignListTotals(
      items as CampaignWithMoneyDisplay[],
      metricsById,
      marginsById,
    );
  const funnelTotals =
    filterTotals?.funnelTotals ?? sumCampaignFunnelTotals(items, metricsById);
  const totalsLabel = filterTotals ? 'Filtered total' : 'Total';

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
          customerNameById,
          ownerEmailById,
        );
        maxWidth = Math.max(maxWidth, columnContentWidth(text, widths[columnId]));
      }
    }

    const totalsText = campaignListTotalsCellDisplayText(
      columnId,
      totals,
      funnelTotals,
      items.length,
      totalsLabel,
    );
    if (totalsText) {
      maxWidth = Math.max(maxWidth, columnContentWidth(totalsText, widths[columnId]));
    }

    widths[columnId] = clampCampaignListColumnWidthPx(columnId, maxWidth);
  }

  return widths;
}

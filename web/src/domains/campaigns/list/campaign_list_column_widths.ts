import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX,
  CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX,
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

const CELL_HORIZONTAL_PADDING_PX = 32;
const HEADER_TOOLS_GUTTER_PX = 28;
const HEADER_SORT_ICON_PX = 12;
const BODY_TOOLS_GUTTER_PX = 28;
const NAME_ROW_MENU_PX = 36;
const NAME_COUNTRY_BADGES_PX = 40;

type ColumnContentWidthOptions = {
  header?: boolean;
  tools?: boolean;
  name?: boolean;
};

function estimateTextWidthPx(text: string): number {
  return Math.ceil(text.length * 7);
}

function columnContentWidth(
  text: string,
  minWidth: number,
  options: ColumnContentWidthOptions = {},
): number {
  let extra = 0;
  if (options.header) {
    extra += HEADER_TOOLS_GUTTER_PX + HEADER_SORT_ICON_PX;
  }
  if (options.tools && !options.header) {
    extra += BODY_TOOLS_GUTTER_PX;
  }
  if (options.name) {
    extra += NAME_ROW_MENU_PX + NAME_COUNTRY_BADGES_PX;
  }
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
    if (columnId === 'select') {
      widths[columnId] = CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX;
      continue;
    }
    const label = CAMPAIGN_LIST_COLUMN_LABELS[columnId];
    widths[columnId] = clampCampaignListColumnWidthPx(
      columnId,
      columnContentWidth(label, CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId], { header: true }),
    );
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
    if (columnId === 'select') {
      widths[columnId] = CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX;
      continue;
    }

    const label = CAMPAIGN_LIST_COLUMN_LABELS[columnId];
    let maxWidth = columnContentWidth(label, widths[columnId], { header: true });

    if (columnId === 'id') {
      for (const campaign of items) {
        maxWidth = Math.max(
          maxWidth,
          columnContentWidth(campaignDisplayId(campaign), widths[columnId], { tools: true }),
        );
      }
    } else if (columnId === 'name') {
      for (const campaign of items) {
        maxWidth = Math.max(
          maxWidth,
          columnContentWidth(campaign.name ?? '', widths[columnId], { name: true, tools: true }),
        );
      }
    } else {
      for (const campaign of items) {
        const text = campaignListMiddleCellText(
          columnId as CampaignListMiddleColumnId,
          campaign,
          metricsById[campaign.id],
          marginsById[campaign.id],
          customerNameById,
          ownerEmailById,
        );
        maxWidth = Math.max(maxWidth, columnContentWidth(text, widths[columnId], { tools: true }));
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
      maxWidth = Math.max(
        maxWidth,
        columnContentWidth(totalsText, widths[columnId], { tools: columnId !== 'select' }),
      );
    }

    widths[columnId] = clampCampaignListColumnWidthPx(columnId, maxWidth);
  }

  return widths;
}

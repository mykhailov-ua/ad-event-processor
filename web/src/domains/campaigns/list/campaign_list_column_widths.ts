import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX,
  CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX,
  CAMPAIGN_LIST_STATUS_COLUMN_WIDTH_PX,
  CAMPAIGN_LIST_STATUS_PROBE_LABEL,
  clampCampaignListColumnWidthPx,
  isCampaignListColumnDraggable,
  type CampaignListColumnId,
  type CampaignListMiddleColumnId,
} from '@/domains/campaigns/list/campaign_list_columns';
import { sortFieldForCampaignColumn } from '@/domains/campaigns/list/campaign_list_sort';
import { campaignDisplayId } from '@/domains/campaigns/list/campaign_display_id';
import { sumCampaignListTotals } from '@/domains/campaigns/list/campaign_list_format';
import { sumCampaignFunnelTotals } from '@/domains/campaigns/list/campaign_list_funnel';
import {
  buildCampaignRowVm,
  buildCampaignRowVmCache,
  campaignListMiddleCellDisplayText,
  type CampaignRowVm,
} from '@/domains/campaigns/list/campaign_list_row_vm';
import { campaignListTotalsCellDisplayText } from '@/domains/campaigns/list/campaign_list_table_totals_display';
import type { CampaignListFilterTotalsView } from '@/domains/campaigns/list/campaign_list_filter_totals';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';

const CELL_HORIZONTAL_PADDING_PX = 32;
const HEADER_DRAG_GRIP_ICON_PX = 12;
const HEADER_SORT_ICON_PX = 12;
const COPY_CONTROL_SLOT_PX = 28;
const COPY_BUTTON_TEXT_GAP_PX = 6;
const NAME_ROW_MENU_PX = 28;
const NAME_ROW_MENU_GAP_PX = 16;
const STATUS_BADGE_HORIZONTAL_PADDING_PX = 16;
const COUNTRY_FLAG_ICON_PX = 18;
const COUNTRY_FLAG_GAP_PX = 2;
const COUNTRY_OVERFLOW_BUTTON_PX = 24;
const COUNTRY_BADGES_MAX_VISIBLE = 3;

type ColumnContentWidthOptions = {
  header?: boolean;
  headerChromePx?: number;
  copyControl?: boolean;
  name?: boolean;
};

function estimateTextWidthPx(text: string): number {
  return Math.ceil(text.length * 7);
}

export function campaignListHeaderChromeWidthPx(columnId: CampaignListColumnId): number {
  let extra = 0;
  if (sortFieldForCampaignColumn(columnId) != null) {
    extra += HEADER_SORT_ICON_PX;
  }
  if (isCampaignListColumnDraggable(columnId)) {
    extra += HEADER_DRAG_GRIP_ICON_PX;
  }
  return extra;
}

function columnContentWidth(
  text: string,
  minWidth: number,
  options: ColumnContentWidthOptions = {}
): number {
  let extra = 0;
  if (options.header) {
    extra += options.headerChromePx ?? 0;
  }
  if (options.copyControl) {
    extra += COPY_CONTROL_SLOT_PX + COPY_BUTTON_TEXT_GAP_PX;
  }
  if (options.name) {
    extra += NAME_ROW_MENU_PX + NAME_ROW_MENU_GAP_PX;
  }
  return Math.max(minWidth, estimateTextWidthPx(text) + CELL_HORIZONTAL_PADDING_PX + extra);
}

export function campaignStatusCellContentWidthPx(label: string, minWidth: number): number {
  return Math.max(
    minWidth,
    estimateTextWidthPx(label) +
      STATUS_BADGE_HORIZONTAL_PADDING_PX +
      CELL_HORIZONTAL_PADDING_PX
  );
}

export function campaignCountriesCellContentWidthPx(
  countryCount: number,
  minWidth: number
): number {
  const visibleCount = Math.min(Math.max(countryCount, 0), COUNTRY_BADGES_MAX_VISIBLE);
  const flagsWidth =
    visibleCount * COUNTRY_FLAG_ICON_PX + Math.max(0, visibleCount - 1) * COUNTRY_FLAG_GAP_PX;
  const overflowWidth =
    countryCount > COUNTRY_BADGES_MAX_VISIBLE
      ? COUNTRY_OVERFLOW_BUTTON_PX + COUNTRY_FLAG_GAP_PX
      : 0;
  return Math.max(
    minWidth,
    flagsWidth + overflowWidth + CELL_HORIZONTAL_PADDING_PX
  );
}

export function campaignListMiddleCellText(
  columnId: CampaignListMiddleColumnId,
  campaign: Campaign,
  metrics: CampaignListMetrics | undefined,
  margin: CampaignMargin | undefined,
  customerNameById: Record<string, string>,
  ownerEmailById: Record<string, string> = {},
  rowVmCache?: ReadonlyMap<string, CampaignRowVm>
): string {
  const vm =
    rowVmCache?.get(campaign.id) ??
    buildCampaignRowVm(campaign, metrics, margin, customerNameById, ownerEmailById, false);
  return campaignListMiddleCellDisplayText(columnId, vm);
}

export function defaultCampaignListColumnWidths(
  columns: ReadonlyArray<CampaignListColumnId>
): Record<CampaignListColumnId, number> {
  const widths = {} as Record<CampaignListColumnId, number>;
  for (const columnId of columns) {
    if (columnId === 'select') {
      widths[columnId] = CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX;
      continue;
    }
    const label = CAMPAIGN_LIST_COLUMN_LABELS[columnId];
    const headerChromePx = campaignListHeaderChromeWidthPx(columnId);
    if (columnId === 'countries') {
      widths[columnId] = clampCampaignListColumnWidthPx(
        columnId,
        Math.max(
          columnContentWidth(label, CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId], {
            header: true,
            headerChromePx,
          }),
          campaignCountriesCellContentWidthPx(COUNTRY_BADGES_MAX_VISIBLE, CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId])
        )
      );
      continue;
    }
    widths[columnId] = clampCampaignListColumnWidthPx(
      columnId,
      columnContentWidth(label, CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX[columnId], {
        header: true,
        headerChromePx,
      })
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
    sumCampaignListTotals(items as CampaignWithMoneyDisplay[], metricsById, marginsById);
  const funnelTotals = filterTotals?.funnelTotals ?? sumCampaignFunnelTotals(items, metricsById);
  const totalsLabel = filterTotals ? 'Filtered total' : 'Total';
  const rowVmCache = buildCampaignRowVmCache(
    items,
    metricsById,
    marginsById,
    customerNameById,
    ownerEmailById
  );

  for (const columnId of columns) {
    if (columnId === 'select') {
      widths[columnId] = CAMPAIGN_LIST_SELECTION_COLUMN_WIDTH_PX;
      continue;
    }

    const label = CAMPAIGN_LIST_COLUMN_LABELS[columnId];
    const headerChromePx = campaignListHeaderChromeWidthPx(columnId);
    let maxWidth = columnContentWidth(label, widths[columnId], {
      header: true,
      headerChromePx,
    });

    if (columnId === 'id') {
      for (const campaign of items) {
        maxWidth = Math.max(
          maxWidth,
          columnContentWidth(campaignDisplayId(campaign), widths[columnId], { copyControl: true })
        );
      }
    } else if (columnId === 'name') {
      for (const campaign of items) {
        maxWidth = Math.max(
          maxWidth,
          columnContentWidth(campaign.name ?? '', widths[columnId], { name: true })
        );
      }
    } else if (columnId === 'countries') {
      for (const campaign of items) {
        maxWidth = Math.max(
          maxWidth,
          campaignCountriesCellContentWidthPx(
            (campaign.target_countries ?? []).length,
            widths[columnId]
          )
        );
      }
    } else if (columnId === 'status') {
      maxWidth = Math.max(
        maxWidth,
        campaignStatusCellContentWidthPx(CAMPAIGN_LIST_STATUS_PROBE_LABEL, widths[columnId])
      );
      for (const campaign of items) {
        const text = campaignListMiddleCellText(
          'status',
          campaign,
          metricsById[campaign.id],
          marginsById[campaign.id],
          customerNameById,
          ownerEmailById,
          rowVmCache
        );
        maxWidth = Math.max(maxWidth, campaignStatusCellContentWidthPx(text, widths[columnId]));
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
          rowVmCache
        );
        maxWidth = Math.max(maxWidth, columnContentWidth(text, widths[columnId]));
      }
    }

    const totalsText = campaignListTotalsCellDisplayText(
      columnId,
      totals,
      funnelTotals,
      items.length,
      totalsLabel
    );
    if (totalsText) {
      maxWidth = Math.max(
        maxWidth,
        columnContentWidth(totalsText, widths[columnId])
      );
    }

    widths[columnId] = clampCampaignListColumnWidthPx(columnId, maxWidth);
  }

  return widths;
}

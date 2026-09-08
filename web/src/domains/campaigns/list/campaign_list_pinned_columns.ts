import {
  isCampaignListPinnedColumn,
  resolveCampaignListColumnWidthPx,
  type CampaignListColumnId,
} from '@/domains/campaigns/list/campaign_list_columns';
import {
  campaignListPinnedTdClass,
  campaignListPinnedThClass,
} from '@/domains/campaigns/list/campaign_list_classes';

export function campaignListPinnedColumnLeftPx(
  columnId: CampaignListColumnId,
  columns: readonly CampaignListColumnId[],
  columnWidths: Readonly<Record<CampaignListColumnId, number>>
): number {
  if (!isCampaignListPinnedColumn(columnId)) {
    return 0;
  }
  let left = 0;
  for (const col of columns) {
    if (col === columnId) {
      break;
    }
    if (isCampaignListPinnedColumn(col)) {
      left += resolveCampaignListColumnWidthPx(col, columnWidths);
    }
  }
  return left;
}

export function campaignListPinnedColumnStyle(
  columnId: CampaignListColumnId,
  columns: readonly CampaignListColumnId[],
  columnWidths: Readonly<Record<CampaignListColumnId, number>>
): { left: string } | undefined {
  if (!isCampaignListPinnedColumn(columnId)) {
    return undefined;
  }
  return { left: `${campaignListPinnedColumnLeftPx(columnId, columns, columnWidths)}px` };
}

export function isCampaignListLastPinnedColumn(
  columnId: CampaignListColumnId,
  columns: readonly CampaignListColumnId[]
): boolean {
  if (!isCampaignListPinnedColumn(columnId)) {
    return false;
  }
  for (let index = columns.length - 1; index >= 0; index -= 1) {
    const col = columns[index];
    if (isCampaignListPinnedColumn(col)) {
      return col === columnId;
    }
  }
  return false;
}

export function campaignListPinnedEdgeWidthPx(
  columns: readonly CampaignListColumnId[],
  columnWidths: Readonly<Record<CampaignListColumnId, number>>
): number {
  let total = 0;
  for (const columnId of columns) {
    if (!isCampaignListPinnedColumn(columnId)) {
      break;
    }
    total += resolveCampaignListColumnWidthPx(columnId, columnWidths);
  }
  return total;
}

export function syncDirectoryPinnedEdgeShadow(
  host: HTMLElement | null | undefined,
  columns: readonly CampaignListColumnId[],
  columnWidths: Readonly<Record<CampaignListColumnId, number>>,
  override?: { columnId: CampaignListColumnId; widthPx: number }
): void {
  if (!host) {
    return;
  }
  const shadow = host.querySelector<HTMLElement>('[data-directory-pinned-edge]');
  if (!shadow) {
    return;
  }
  shadow.style.left = `${campaignListPinnedEdgeWidthPx(
    columns,
    override
      ? {
          ...columnWidths,
          [override.columnId]: override.widthPx,
        }
      : columnWidths
  )}px`;
}

export function campaignListPinnedCellClassName(
  columnId: CampaignListColumnId,
  columns: readonly CampaignListColumnId[],
  variant: 'header' | 'body' | 'footer'
): string {
  if (!isCampaignListPinnedColumn(columnId)) {
    return '';
  }
  return variant === 'header' ? campaignListPinnedThClass : campaignListPinnedTdClass;
}

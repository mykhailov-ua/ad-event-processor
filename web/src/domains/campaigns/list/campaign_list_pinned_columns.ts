import {
  isCampaignListPinnedColumn,
  resolveCampaignListColumnWidthPx,
  type CampaignListColumnId,
} from '@/domains/campaigns/list/campaign_list_columns';
import {
  campaignListPinnedEdgeClass,
  campaignListPinnedTdClass,
  campaignListPinnedThClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import { cn } from '@/lib/utils';

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

export function campaignListPinnedCellClassName(
  columnId: CampaignListColumnId,
  columns: readonly CampaignListColumnId[],
  variant: 'header' | 'body' | 'footer'
): string {
  if (!isCampaignListPinnedColumn(columnId)) {
    return '';
  }
  const last = isCampaignListLastPinnedColumn(columnId, columns);
  return cn(
    variant === 'header' ? campaignListPinnedThClass : campaignListPinnedTdClass,
    last && campaignListPinnedEdgeClass
  );
}

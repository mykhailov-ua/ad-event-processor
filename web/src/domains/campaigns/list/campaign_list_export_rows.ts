import type { Campaign } from '@/api/types';
import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  isCampaignListMiddleColumnId,
  type CampaignListColumnId,
  type CampaignListDataColumnId,
} from '@/domains/campaigns/list/campaign_list_columns';
import {
  buildCampaignRowVm,
  campaignListMiddleCellDisplayText,
  type CampaignRowVm,
} from '@/domains/campaigns/list/campaign_list_row_vm';
import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { CampaignMargin } from '@/api/types';

export type CampaignListExportRow = {
  campaign: Campaign;
  vm: CampaignRowVm;
};

export function exportableCampaignListColumns(
  columns: ReadonlyArray<CampaignListColumnId>,
): CampaignListDataColumnId[] {
  return columns.filter((columnId): columnId is CampaignListDataColumnId => columnId !== 'select');
}

export function campaignListExportCellValue(
  columnId: CampaignListDataColumnId,
  vm: CampaignRowVm,
): string {
  if (columnId === 'id') {
    return vm.displayId;
  }
  if (columnId === 'name') {
    return vm.rawName;
  }
  if (isCampaignListMiddleColumnId(columnId)) {
    return campaignListMiddleCellDisplayText(columnId, vm);
  }
  return '';
}

export function buildCampaignListExportRows(
  campaigns: Campaign[],
  columns: ReadonlyArray<CampaignListDataColumnId>,
  metricsById: Record<string, CampaignListMetrics>,
  marginsById: Record<string, CampaignMargin>,
  customerNameById: Record<string, string>,
  ownerEmailById: Record<string, string>,
): CampaignListExportRow[] {
  if (columns.length === 0) {
    return [];
  }
  return campaigns.map((campaign) => ({
    campaign,
    vm: buildCampaignRowVm(
      campaign,
      metricsById[campaign.id],
      marginsById[campaign.id],
      customerNameById,
      ownerEmailById,
      false,
    ),
  }));
}

function csvEscape(value: string): string {
  return `"${value.replaceAll('"', '""')}"`;
}

export function buildCampaignListExportCsv(
  columns: ReadonlyArray<CampaignListDataColumnId>,
  rows: ReadonlyArray<CampaignListExportRow>,
): string {
  const header = columns.map((columnId) => CAMPAIGN_LIST_COLUMN_LABELS[columnId]);
  const lines = rows.map((row) =>
    columns.map((columnId) => csvEscape(campaignListExportCellValue(columnId, row.vm))).join(','),
  );
  return [header.join(','), ...lines].join('\n');
}

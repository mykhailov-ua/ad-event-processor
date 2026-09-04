import { exportCampaign, exportCampaignsBatch, listCampaigns } from '@/api/campaigns_api';
import type { Campaign } from '@/api/types';
import type { CampaignListDataColumnId } from '@/domains/campaigns/list/campaign_list_columns';
import {
  buildCampaignListExportCsv,
  type CampaignListExportRow,
} from '@/domains/campaigns/list/campaign_list_export_rows';
export { formatCampaignListExportToast } from '@/domains/campaigns/list/campaign_list_export_toast';
import type { CampaignListFilterQuery } from '@/domains/campaigns/list/campaigns_list_query';
import {
  CAMPAIGN_LIST_EXPORT_BATCH_CHUNK_SIZE,
  CAMPAIGN_LIST_EXPORT_MAX_ROWS,
} from '@/domains/campaigns/list/campaign_list_limits';

const CAMPAIGN_LIST_EXPORT_PAGE_SIZE = 1000;

export type CampaignListExportDataset = {
  items: Campaign[];
  matchedTotal: number;
  truncated: boolean;
};

export async function listAllCampaignsForFilter(
  filter: CampaignListFilterQuery,
  signal?: AbortSignal,
): Promise<CampaignListExportDataset> {
  const items: Campaign[] = [];
  let offset = 0;
  let matchedTotal = Number.POSITIVE_INFINITY;

  while (offset < matchedTotal && items.length < CAMPAIGN_LIST_EXPORT_MAX_ROWS) {
    const page = await listCampaigns(
      {
        ...filter,
        limit: CAMPAIGN_LIST_EXPORT_PAGE_SIZE,
        offset,
        sort: 'name',
        order: 'asc',
      },
      signal,
    );
    matchedTotal = page.total;
    if (page.items.length === 0) {
      break;
    }
    items.push(...page.items);
    offset += page.items.length;
    if (items.length >= CAMPAIGN_LIST_EXPORT_MAX_ROWS) {
      break;
    }
  }

  const resolvedTotal = Number.isFinite(matchedTotal) ? matchedTotal : items.length;
  return {
    items,
    matchedTotal: resolvedTotal,
    truncated: items.length < resolvedTotal,
  };
}

export function exportCampaignRowsCsv(
  columns: ReadonlyArray<CampaignListDataColumnId>,
  rows: ReadonlyArray<CampaignListExportRow>,
): void {
  if (columns.length === 0 || rows.length === 0) {
    return;
  }
  const csv = buildCampaignListExportCsv(columns, rows);
  const blob = new Blob([csv], {
    type: 'text/csv;charset=utf-8',
  });
  downloadBlob(blob, 'campaigns-export.csv');
}

async function runInChunks<T>(
  ids: string[],
  chunkSize: number,
  runner: (chunk: string[]) => Promise<T>,
): Promise<T[]> {
  const results: T[] = [];
  for (let offset = 0; offset < ids.length; offset += chunkSize) {
    results.push(await runner(ids.slice(offset, offset + chunkSize)));
  }
  return results;
}

export async function exportCampaignBundles(campaignIds: string[]): Promise<void> {
  if (campaignIds.length === 0) {
    return;
  }

  if (campaignIds.length === 1) {
    const bundle = await exportCampaign(campaignIds[0]!);
    const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: 'application/json' });
    downloadBlob(blob, `campaign-${campaignIds[0]}.json`);
    return;
  }

  const bundles: Record<string, unknown> = {};
  const chunkResults = await runInChunks(
    campaignIds,
    CAMPAIGN_LIST_EXPORT_BATCH_CHUNK_SIZE,
    (chunk) => exportCampaignsBatch(chunk),
  );
  for (const response of chunkResults) {
    Object.assign(bundles, response.items);
  }
  const blob = new Blob([JSON.stringify(bundles, null, 2)], { type: 'application/json' });
  downloadBlob(blob, 'campaigns-export.json');
}

function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

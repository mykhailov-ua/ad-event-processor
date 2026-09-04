import {
  bulkCampaignAction,
  summarizeCampaignBulkResults,
  type CampaignBulkAction,
} from '@/api/campaigns_api';
import { CAMPAIGN_LIST_BULK_CHUNK_SIZE } from '@/domains/campaigns/list/campaign_list_limits';

export type CampaignBulkActionResult = {
  succeeded: string[];
  failed: { id: string; error: string }[];
};

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

async function bulkCampaignActionChunked(
  action: CampaignBulkAction,
  campaignIds: string[],
): Promise<CampaignBulkActionResult> {
  if (campaignIds.length === 0) {
    return { succeeded: [], failed: [] };
  }

  const chunkResults = await runInChunks(campaignIds, CAMPAIGN_LIST_BULK_CHUNK_SIZE, (chunk) =>
    bulkCampaignAction({ action, campaign_ids: chunk }),
  );

  const succeeded: string[] = [];
  const failed: { id: string; error: string }[] = [];
  for (const response of chunkResults) {
    const summary = summarizeCampaignBulkResults(response.results);
    for (const row of summary.succeeded) {
      succeeded.push(row.id);
    }
    for (const row of summary.failed) {
      failed.push({ id: row.id, error: row.error_code ?? 'BULK_FAILED' });
    }
  }
  return { succeeded, failed };
}

export async function bulkPauseOrResumeCampaigns(
  action: Extract<CampaignBulkAction, 'pause' | 'resume'>,
  campaignIds: string[],
): Promise<CampaignBulkActionResult> {
  return bulkCampaignActionChunked(action, campaignIds);
}

export async function archiveCampaigns(campaignIds: string[]): Promise<CampaignBulkActionResult> {
  return bulkCampaignActionChunked('archive', campaignIds);
}

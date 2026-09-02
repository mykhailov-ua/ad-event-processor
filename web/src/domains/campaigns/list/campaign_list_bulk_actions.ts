import {
  bulkCampaignAction,
  CAMPAIGN_BULK_ACTION_MAX_IDS,
  patchCampaign,
  summarizeCampaignBulkResults,
  type CampaignBulkAction,
} from '@/api/campaigns_api';

export type CampaignBulkArchiveResult = {
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

export async function bulkPauseOrResumeCampaigns(
  action: CampaignBulkAction,
  campaignIds: string[],
): Promise<{ succeeded: string[]; failed: { id: string; error: string }[] }> {
  if (campaignIds.length === 0) {
    return { succeeded: [], failed: [] };
  }

  const chunkResults = await runInChunks(campaignIds, CAMPAIGN_BULK_ACTION_MAX_IDS, (chunk) =>
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

export async function archiveCampaigns(campaignIds: string[]): Promise<CampaignBulkArchiveResult> {
  if (campaignIds.length === 0) {
    return { succeeded: [], failed: [] };
  }

  const settled = await Promise.allSettled(
    campaignIds.map(async (id) => {
      await patchCampaign(id, { status: 'ARCHIVED' });
      return id;
    }),
  );

  const succeeded: string[] = [];
  const failed: { id: string; error: string }[] = [];
  settled.forEach((result, index) => {
    const id = campaignIds[index] ?? '';
    if (result.status === 'fulfilled') {
      succeeded.push(result.value);
      return;
    }
    const message =
      result.reason instanceof Error ? result.reason.message : String(result.reason);
    failed.push({ id, error: message });
  });
  return { succeeded, failed };
}

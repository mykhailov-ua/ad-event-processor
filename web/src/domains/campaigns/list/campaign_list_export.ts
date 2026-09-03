import { listCampaigns } from '@/api/campaigns_api';
import { exportCampaign } from '@/api/campaigns_api';
import type { Campaign } from '@/api/types';
import type { CampaignListFilterQuery } from '@/domains/campaigns/list/campaigns_list_query';

const CAMPAIGN_LIST_EXPORT_PAGE_SIZE = 1000;
const CAMPAIGN_LIST_EXPORT_MAX_ROWS = 5000;

export async function listAllCampaignsForFilter(
  filter: CampaignListFilterQuery,
  signal?: AbortSignal,
): Promise<Campaign[]> {
  const items: Campaign[] = [];
  let offset = 0;
  let total = Number.POSITIVE_INFINITY;

  while (offset < total && items.length < CAMPAIGN_LIST_EXPORT_MAX_ROWS) {
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
    total = page.total;
    if (page.items.length === 0) {
      break;
    }
    items.push(...page.items);
    offset += page.items.length;
    if (items.length >= CAMPAIGN_LIST_EXPORT_MAX_ROWS) {
      break;
    }
  }

  return items;
}

export function exportCampaignRowsCsv(
  items: Campaign[],
  customerNameById: Record<string, string>,
): void {
  const header = ['id', 'name', 'status', 'customer', 'budget', 'spend'];
  const lines = items.map((campaign) =>
    [
      campaign.id,
      campaign.name,
      campaign.status,
      customerNameById[campaign.customer_id] ?? campaign.customer_id,
      campaign.budget_limit,
      campaign.current_spend,
    ]
      .map((value) => `"${String(value).replaceAll('"', '""')}"`)
      .join(','),
  );
  const blob = new Blob([[header.join(','), ...lines].join('\n')], {
    type: 'text/csv;charset=utf-8',
  });
  downloadBlob(blob, 'campaigns-export.csv');
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
  for (const campaignId of campaignIds) {
    bundles[campaignId] = await exportCampaign(campaignId);
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

import { exportCampaign } from '@/api/campaigns_api';
import type { Campaign } from '@/api/types';

export function exportVisibleRowsCsv(
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

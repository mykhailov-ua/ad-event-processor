import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign } from '@/api/types';
import {
  visibleCampaignListColumns,
  defaultCampaignListColumnPrefs,
} from '@/domains/campaigns/list/campaign_list_columns';
import { seedDeterministicUuid } from '@/lib/uuid';

export function buildLargeCampaignListFixture(rowCount: number): {
  items: Campaign[];
  metricsById: Record<string, CampaignListMetrics>;
  customerNameById: Record<string, string>;
  columns: ReturnType<typeof visibleCampaignListColumns>;
} {
  const customerId = seedDeterministicUuid('customer', 1);
  const items: Campaign[] = [];
  const metricsById: Record<string, CampaignListMetrics> = {};

  for (let i = 0; i < rowCount; i += 1) {
    const id = seedDeterministicUuid('campaign', i + 1);
    items.push({
      id,
      name: `Campaign ${i + 1} | US | Native | Long label for width probe ${i}`,
      customer_id: customerId,
      status: i % 3 === 0 ? 'PAUSED' : 'ACTIVE',
      owner_user_id: seedDeterministicUuid('user', (i % 5) + 1),
      target_countries: ['US', 'DE'],
    } as Campaign);
    metricsById[id] = {
      clicks: 1_000 + i * 17,
      impressions: 10_000 + i * 113,
      conversions: 10 + (i % 40),
      revenue_micro: 5_000_000_000 + i * 1_000_000,
      cost_micro: 3_000_000_000 + i * 900_000,
      epc_micro: 4_500_000 + i * 1_000,
      ctr_pct: 1.2 + (i % 10) * 0.1,
      cr_pct: 0.8 + (i % 5) * 0.05,
      roi_pct: 12 + (i % 20),
    };
  }

  const prefs = defaultCampaignListColumnPrefs();
  const columns = visibleCampaignListColumns(prefs);

  return {
    items,
    metricsById,
    customerNameById: { [customerId]: 'Fixture Buyer GmbH' },
    columns,
  };
}

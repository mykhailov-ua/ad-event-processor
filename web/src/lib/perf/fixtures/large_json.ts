import {
  defaultCampaignListColumnPrefs,
  serializeCampaignListColumnPrefs,
} from '@/domains/campaigns/list/campaign_list_columns';

/** ~5 KiB localStorage-shaped campaign column prefs payload. */
export function buildLargeCampaignColumnPrefsJson(): string {
  const prefs = defaultCampaignListColumnPrefs();
  return serializeCampaignListColumnPrefs(prefs);
}

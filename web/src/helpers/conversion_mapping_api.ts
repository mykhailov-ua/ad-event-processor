import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

/** One inbound affiliate status mapped to a report goal and payout. */
export type ConversionMappingRow = {
  inbound_status: string;
  goal_name: string;
  payout_micro: number;
};

/** Affiliate network status preset from bundled integration schemas. */
export type AffiliateStatusPreset = {
  name: string;
  statuses: Array<{
    inbound_status: string;
    goal_name: string;
  }>;
};

/**
 * Load campaign conversion type to payout mappings.
 */
export async function fetchConversionMappings(campaignId: string): Promise<ConversionMappingRow[]> {
  const { data } = await api(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/conversion-mappings`
  );
  const body = data as { mappings?: ConversionMappingRow[] } | null | undefined;
  return body?.mappings ?? [];
}

/**
 * Replace all conversion mappings for a campaign.
 */
export async function replaceConversionMappings(
  campaignId: string,
  mappings: ConversionMappingRow[]
): Promise<ConversionMappingRow[]> {
  const res = await apiConfirmed(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/conversion-mappings`,
    {
      method: 'PUT',
      body: JSON.stringify({ mappings }),
    }
  );
  const body = res.data as { mappings?: ConversionMappingRow[] } | null | undefined;
  return body?.mappings ?? [];
}

/**
 * List bundled affiliate status presets for mapping table seed rows.
 */
export async function fetchAffiliateStatusPresets(): Promise<AffiliateStatusPreset[]> {
  const { data } = await api('/api/v1/integration/affiliate-status-presets');
  return (data as AffiliateStatusPreset[] | null | undefined) ?? [];
}

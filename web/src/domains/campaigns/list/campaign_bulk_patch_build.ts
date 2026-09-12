import type { CampaignBulkPatchFields } from '@/api/campaigns_types';

export type CampaignBulkPatchFieldKey = keyof CampaignBulkPatchFields;

export type CampaignBulkPatchDraft = {
  target_url: string;
  budget_limit: string;
  pacing_mode: string;
  target_countries: string;
  status: string;
  timezone: string;
  referrer_filter: string;
  fallback_click_url: string;
  budget_failover_mode: string;
};

export type CampaignBulkPatchEnabled = Record<CampaignBulkPatchFieldKey, boolean>;

export const EMPTY_CAMPAIGN_BULK_PATCH_DRAFT: CampaignBulkPatchDraft = {
  target_url: '',
  budget_limit: '',
  pacing_mode: '',
  target_countries: '',
  status: '',
  timezone: '',
  referrer_filter: '',
  fallback_click_url: '',
  budget_failover_mode: '',
};

export const EMPTY_CAMPAIGN_BULK_PATCH_ENABLED: CampaignBulkPatchEnabled = {
  target_url: false,
  budget_limit: false,
  budget_limit_micro: false,
  pacing_mode: false,
  daily_budget_micro: false,
  target_countries: false,
  status: false,
  timezone: false,
  referrer_filter: false,
  fallback_click_url: false,
  budget_failover_mode: false,
};

function parseTargetCountries(raw: string): string[] {
  const parts = raw
    .split(/[,\s]+/)
    .map((part) => part.trim().toUpperCase())
    .filter((part) => part.length > 0);
  return [...new Set(parts)];
}

export function buildCampaignBulkPatchPayload(
  draft: CampaignBulkPatchDraft,
  enabled: CampaignBulkPatchEnabled
): CampaignBulkPatchFields | undefined {
  const patch: CampaignBulkPatchFields = {};

  if (enabled.target_url) {
    patch.target_url = draft.target_url.trim();
  }
  if (enabled.budget_limit) {
    patch.budget_limit = draft.budget_limit.trim();
  }
  if (enabled.pacing_mode) {
    patch.pacing_mode = draft.pacing_mode.trim();
  }
  if (enabled.target_countries) {
    patch.target_countries = parseTargetCountries(draft.target_countries);
  }
  if (enabled.status) {
    patch.status = draft.status.trim().toLowerCase();
  }
  if (enabled.timezone) {
    patch.timezone = draft.timezone.trim();
  }
  if (enabled.referrer_filter) {
    patch.referrer_filter = draft.referrer_filter.trim();
  }
  if (enabled.fallback_click_url) {
    patch.fallback_click_url = draft.fallback_click_url.trim();
  }
  if (enabled.budget_failover_mode) {
    patch.budget_failover_mode = draft.budget_failover_mode.trim().toLowerCase();
  }

  return Object.keys(patch).length > 0 ? patch : undefined;
}

export function validateCampaignBulkPatchDraft(
  draft: CampaignBulkPatchDraft,
  enabled: CampaignBulkPatchEnabled
): string | undefined {
  const patch = buildCampaignBulkPatchPayload(draft, enabled);
  if (!patch) {
    return 'Enable at least one field to update';
  }
  if (enabled.target_url && !patch.target_url) {
    return 'Target URL is required when enabled';
  }
  if (enabled.budget_limit && !patch.budget_limit) {
    return 'Budget limit is required when enabled';
  }
  if (enabled.pacing_mode && !patch.pacing_mode) {
    return 'Pacing mode is required when enabled';
  }
  if (enabled.target_countries && (patch.target_countries?.length ?? 0) === 0) {
    return 'Enter at least one country code when countries are enabled';
  }
  if (enabled.status && !patch.status) {
    return 'Status is required when enabled';
  }
  if (enabled.timezone && !patch.timezone) {
    return 'Timezone is required when enabled';
  }
  if (enabled.fallback_click_url && !patch.fallback_click_url) {
    return 'Fallback click URL is required when enabled';
  }
  if (enabled.budget_failover_mode && !patch.budget_failover_mode) {
    return 'Budget failover mode is required when enabled';
  }
  return undefined;
}

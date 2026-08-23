import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type FraudPolicyPreset = {
  name: string;
  pass: number;
  suspect: number;
  ivt: number;
  block: number;
  updated_at?: string;
};

export type FraudSensitivityPreset =
  | 'conservative'
  | 'balanced'
  | 'aggressive'
  | 'gray_market'
  | 'social_in_app';

export type CampaignFraudConfig = {
  campaign_id: string;
  fraud_threshold_pass: number;
  fraud_threshold_suspect: number;
  fraud_threshold_ivt: number;
  fraud_threshold_block: number;
  ghost_ivt_enabled: boolean;
  behavior_flags?: number;
};

export type PatchCampaignFraudRequest = {
  preset?: FraudSensitivityPreset;
  fraud_threshold_pass?: number;
  fraud_threshold_suspect?: number;
  fraud_threshold_ivt?: number;
  fraud_threshold_block?: number;
  ghost_ivt_enabled?: boolean;
};

export type CampaignFraudPreview = {
  campaign_id: string;
  affected_ips_7d: number;
  sample_size: number;
  by_tier: {
    suspect: number;
    ivt: number;
    block: number;
  };
  disclaimer: string;
};

export type PreviewCampaignFraudRequest = {
  preset?: FraudSensitivityPreset;
  fraud_threshold_pass?: number;
  fraud_threshold_suspect?: number;
  fraud_threshold_ivt?: number;
  fraud_threshold_block?: number;
};

export async function fetchCampaignFraudConfig(
  campaignId: string
): Promise<CampaignFraudConfig | null> {
  const res = await api<CampaignFraudConfig>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/fraud`
  );
  return res.data ?? null;
}

export async function patchCampaignFraudConfig(
  campaignId: string,
  body: PatchCampaignFraudRequest
): Promise<CampaignFraudConfig | null> {
  const res = await apiConfirmed<CampaignFraudConfig>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/fraud`,
    {
      method: 'PATCH',
      body: JSON.stringify(body),
    }
  );
  return res.data ?? null;
}

export async function previewCampaignFraudImpact(
  campaignId: string,
  body: PreviewCampaignFraudRequest
): Promise<CampaignFraudPreview | null> {
  const res = await api<CampaignFraudPreview>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/fraud/preview`,
    {
      method: 'POST',
      body: JSON.stringify(body),
    }
  );
  return res.data ?? null;
}

export const FRAUD_PRESET_OPTIONS: Array<{
  id: FraudSensitivityPreset;
  label: string;
  description: string;
}> = [
  {
    id: 'conservative',
    label: 'Conservative',
    description: 'Fewer blocks; higher score bands before action.',
  },
  {
    id: 'balanced',
    label: 'Balanced',
    description: 'Platform defaults for pass / suspect / ivt / block.',
  },
  {
    id: 'aggressive',
    label: 'Aggressive',
    description: 'More blocks; lower score bands before action.',
  },
  {
    id: 'gray_market',
    label: 'Gray market (GMA)',
    description:
      'Aggressive fraud tiers plus safe page, L2 attestation, L1/L1.5/TLS blocks, and signed links.',
  },
  {
    id: 'social_in_app',
    label: 'Social in-app',
    description:
      'Balanced tiers, mobile-only conn policy, L1.5/TLS blocks; TLS safe-view skipped for FB/TikTok/Instagram WebView UA.',
  },
];

export async function fetchFraudPresets(): Promise<FraudPolicyPreset[]> {
  const res = await api<FraudPolicyPreset[]>('/api/v1/fraud/presets');
  if (Array.isArray(res.data) && res.data.length > 0) {
    return res.data;
  }
  return FRAUD_PRESET_OPTIONS.map((opt) => {
    const aggressive = opt.id === 'aggressive' || opt.id === 'gray_market';
    const conservative = opt.id === 'conservative';
    return {
      name: opt.id,
      pass: conservative ? 40 : aggressive ? 20 : 30,
      suspect: conservative ? 70 : aggressive ? 45 : 60,
      ivt: conservative ? 90 : aggressive ? 65 : 80,
      block: conservative ? 100 : aggressive ? 85 : 100,
    };
  });
}

export type FraudManualLabel = {
  ip_hash: string;
  label: number;
  reason?: string;
  source?: string;
  created_at?: string;
};

export type FraudManualLabelRequest = {
  ip_hash: string;
  label: number;
  reason?: string;
};

const IP_HASH_RE = /^[0-9a-fA-F]{32}$/;

export function isValidFraudIPHash(value: string): boolean {
  return IP_HASH_RE.test(value.trim());
}

export async function fetchFraudLabels(
  customerId: string,
  limit = 50
): Promise<FraudManualLabel[]> {
  const qs = new URLSearchParams({
    customer_id: customerId,
    limit: String(limit),
  });
  const res = await api<FraudManualLabel[]>(`/api/v1/fraud/labels?${qs.toString()}`);
  return Array.isArray(res.data) ? res.data : [];
}

export async function postFraudLabel(
  customerId: string,
  body: FraudManualLabelRequest
): Promise<void> {
  const qs = new URLSearchParams({ customer_id: customerId });
  await apiConfirmed(`/api/v1/fraud/labels?${qs.toString()}`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

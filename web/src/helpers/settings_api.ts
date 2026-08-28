import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type StripeConfig = {
  enabled?: boolean;
  secret_key?: string;
  webhook_secret?: string;
  checkout_success_url?: string;
  checkout_cancel_url?: string;
};

export type PlatformConfig = {
  tracking_domain?: string;
  default_currency?: string;
  timezone?: string;
  ingress_schema?: string;
  telemetry_enabled?: boolean;
  stripe?: StripeConfig;
  profile?: string;
  edge_xdp?: boolean;
  edge_expose_click?: boolean;
  edge_expose_openrtb?: boolean;
  network_interface?: string;
};

export type MaskedSecrets = {
  stripe_secret_key?: string;
  stripe_webhook_secret?: string;
};

export type PlatformSettingsView = {
  config?: PlatformConfig;
  secrets?: MaskedSecrets;
  restart_required?: string[];
  click_url_template?: string;
  openrtb_endpoint_template?: string;
  bootstrap_complete?: boolean;
};

export type PlatformSettingsPatch = {
  tracking_domain?: string;
  default_currency?: string;
  timezone?: string;
  ingress_schema?: string;
  telemetry_enabled?: boolean;
  stripe?: {
    enabled?: boolean;
    secret_key?: string;
    webhook_secret?: string;
    checkout_success_url?: string;
    checkout_cancel_url?: string;
  };
  profile?: string;
  edge_xdp?: boolean;
  edge_expose_click?: boolean;
  edge_expose_openrtb?: boolean;
  network_interface?: string;
};

export type PlatformApplyResponse = {
  written_path?: string;
};

export type LicenseStatus = {
  deployment_id?: string;
  state?: string;
  valid_until?: string;
  host_fingerprint?: string;
  hwid_v2?: string;
  hwid_match?: boolean;
  days_to_expiry?: number;
  plan_code?: string;
  max_rps?: number;
  upgrade_plan_code?: string;
  trial_self_serve_url?: string;
  pilot_valid_days?: number;
  support_url?: string;
};

export async function fetchPlatformSettings(signal?: AbortSignal): Promise<PlatformSettingsView> {
  const result = await api<PlatformSettingsView>('/api/v1/settings/platform', { signal });
  return result.data ?? {};
}

export async function patchPlatformSettings(
  body: PlatformSettingsPatch
): Promise<PlatformSettingsView> {
  const result = await apiConfirmed<PlatformSettingsView>('/api/v1/settings/platform', {
    method: 'PATCH',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('platform settings save failed');
  }
  return result.data ?? {};
}

export async function applyPlatformSettings(
  installRoot?: string
): Promise<PlatformApplyResponse> {
  const result = await apiConfirmed<PlatformApplyResponse>('/api/v1/settings/platform/apply', {
    method: 'POST',
    body: JSON.stringify(installRoot ? { install_root: installRoot } : {}),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('platform apply failed');
  }
  return result.data ?? {};
}

export async function fetchLicenseStatus(signal?: AbortSignal): Promise<LicenseStatus> {
  const result = await api<LicenseStatus>('/api/v1/license/status', { signal });
  return result.data ?? {};
}

export async function applyLicenseToken(token: string): Promise<LicenseStatus> {
  const result = await apiConfirmed<LicenseStatus>('/api/v1/license/apply', {
    method: 'POST',
    body: JSON.stringify({ token }),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('license apply failed');
  }
  return result.data ?? {};
}

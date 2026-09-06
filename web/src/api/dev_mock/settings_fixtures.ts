function isRecord(value: unknown): value is Record<string, unknown> {
  return value != null && typeof value === 'object' && !Array.isArray(value);
}

function maskSecret(value: string): string {
  const trimmed = value.trim();
  if (trimmed.length <= 8) {
    return '********';
  }
  return `${trimmed.slice(0, 4)}...${trimmed.slice(-4)}`;
}

const CONFIG_PATCH_KEYS = new Set([
  'tracking_domain',
  'default_currency',
  'timezone',
  'ingress_schema',
  'telemetry_enabled',
  'profile',
  'edge_xdp',
  'edge_expose_click',
  'edge_expose_openrtb',
  'network_interface',
]);

function createDevMockPlatformSettings(): Record<string, unknown> {
  return {
    bootstrap_complete: true,
    restart_required: false,
    click_url_template: 'https://track.local/click/{campaign_id}',
    openrtb_endpoint_template: 'https://track.local/openrtb/auction',
    config: {
      tracking_domain: 'track.local',
      default_currency: 'USD',
      timezone: 'UTC',
      ingress_schema: 'ad_event_processor_native',
      telemetry_enabled: true,
      profile: 'single_vps',
      edge_xdp: false,
      edge_expose_click: false,
      edge_expose_openrtb: false,
      network_interface: 'eth0',
      stripe: {
        enabled: false,
      },
    },
    secrets: {
      stripe_secret_key: 'sk_dev_redacted',
      stripe_webhook_secret: 'whsec_dev_redacted',
    },
  };
}

let devMockPlatformSettings = createDevMockPlatformSettings();

export function devMockPlatformSettingsView(): Record<string, unknown> {
  return structuredClone(devMockPlatformSettings);
}

export function devMockResetPlatformSettingsForTests(): void {
  devMockPlatformSettings = createDevMockPlatformSettings();
}

export function devMockPatchPlatformSettings(
  patch: Record<string, unknown>
): Record<string, unknown> {
  const next = structuredClone(devMockPlatformSettings) as Record<string, unknown>;
  const config = isRecord(next.config) ? { ...next.config } : {};
  const secrets = isRecord(next.secrets) ? { ...next.secrets } : {};

  if (isRecord(patch.config)) {
    Object.assign(config, patch.config);
  }

  if (isRecord(patch.stripe)) {
    const stripeConfig = isRecord(config.stripe) ? { ...config.stripe } : {};
    if (typeof patch.stripe.enabled === 'boolean') {
      stripeConfig.enabled = patch.stripe.enabled;
    }
    config.stripe = stripeConfig;
    if (typeof patch.stripe.secret_key === 'string' && patch.stripe.secret_key.trim()) {
      secrets.stripe_secret_key = maskSecret(patch.stripe.secret_key);
    }
    if (typeof patch.stripe.webhook_secret === 'string' && patch.stripe.webhook_secret.trim()) {
      secrets.stripe_webhook_secret = maskSecret(patch.stripe.webhook_secret);
    }
  }

  for (const [key, value] of Object.entries(patch)) {
    if (key === 'config' || key === 'stripe') {
      continue;
    }
    if (CONFIG_PATCH_KEYS.has(key)) {
      config[key] = value;
      continue;
    }
    next[key] = value;
  }

  next.config = config;
  next.secrets = secrets;
  devMockPlatformSettings = next;
  return devMockPlatformSettingsView();
}

export function devMockApplyPlatformSettings(installRoot?: string): { written_path: string } {
  const root = installRoot?.trim() || '/var/lib/ad-event-processor';
  return {
    written_path: `${root.replace(/\/$/, '')}/platform_config.json`,
  };
}

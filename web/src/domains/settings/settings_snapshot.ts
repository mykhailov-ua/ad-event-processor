export type PlatformConfigSnapshot = {
  trackingDomain: string;
  defaultCurrency: string;
  timezone: string;
  ingressSchema: string;
  telemetryEnabled: boolean | undefined;
  profile: string;
  edgeXdp: boolean | undefined;
  edgeExposeClick: boolean | undefined;
  edgeExposeOpenRTB: boolean | undefined;
  networkInterface: string;
  stripeEnabled: boolean | undefined;
  stripeCheckoutSuccessUrl: string;
  stripeCheckoutCancelUrl: string;
};

export type PlatformSecretsSnapshot = {
  stripeSecretKey: string;
  stripeWebhookSecret: string;
};

export type PlatformSettingsSnapshot = {
  bootstrapComplete: boolean;
  restartPending: string[];
  clickUrlTemplate: string;
  openRtbEndpointTemplate: string;
  config: PlatformConfigSnapshot;
  secrets: PlatformSecretsSnapshot;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return value != null && typeof value === 'object' && !Array.isArray(value);
}

function readString(record: Record<string, unknown>, key: string): string {
  const value = record[key];
  return typeof value === 'string' ? value.trim() : '';
}

function readBool(record: Record<string, unknown>, key: string): boolean | undefined {
  const value = record[key];
  if (typeof value === 'boolean') {
    return value;
  }
  if (value === 'true') {
    return true;
  }
  if (value === 'false') {
    return false;
  }
  return undefined;
}

function readRestartPending(payload: Record<string, unknown>): string[] {
  const value = payload.restart_required;
  if (Array.isArray(value)) {
    return value.filter((entry): entry is string => typeof entry === 'string' && entry.trim() !== '');
  }
  if (value === true || value === 'true') {
    return ['configuration'];
  }
  return [];
}

function parseConfig(value: unknown): PlatformConfigSnapshot {
  const record = isRecord(value) ? value : {};
  const stripe = isRecord(record.stripe) ? record.stripe : {};
  return {
    trackingDomain: readString(record, 'tracking_domain'),
    defaultCurrency: readString(record, 'default_currency'),
    timezone: readString(record, 'timezone'),
    ingressSchema: readString(record, 'ingress_schema'),
    telemetryEnabled: readBool(record, 'telemetry_enabled'),
    profile: readString(record, 'profile'),
    edgeXdp: readBool(record, 'edge_xdp'),
    edgeExposeClick: readBool(record, 'edge_expose_click'),
    edgeExposeOpenRTB: readBool(record, 'edge_expose_openrtb'),
    networkInterface: readString(record, 'network_interface'),
    stripeEnabled: readBool(stripe, 'enabled'),
    stripeCheckoutSuccessUrl: readString(stripe, 'checkout_success_url'),
    stripeCheckoutCancelUrl: readString(stripe, 'checkout_cancel_url'),
  };
}

function parseSecrets(value: unknown): PlatformSecretsSnapshot {
  const record = isRecord(value) ? value : {};
  return {
    stripeSecretKey: readString(record, 'stripe_secret_key'),
    stripeWebhookSecret: readString(record, 'stripe_webhook_secret'),
  };
}

export function parsePlatformSettingsSnapshot(
  payload: Record<string, unknown>,
): PlatformSettingsSnapshot {
  const bootstrapValue = payload.bootstrap_complete;
  const bootstrapComplete = bootstrapValue === true || bootstrapValue === 'true';
  return {
    bootstrapComplete,
    restartPending: readRestartPending(payload),
    clickUrlTemplate: readString(payload, 'click_url_template'),
    openRtbEndpointTemplate: readString(payload, 'openrtb_endpoint_template'),
    config: parseConfig(payload.config),
    secrets: parseSecrets(payload.secrets),
  };
}

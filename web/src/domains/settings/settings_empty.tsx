import type { ReactNode } from 'react';

import { formatSettingsDisplayValue, usesSettingsDisplayValue } from '@/lib/settings_display_values';

const EMPTY_COPY = {
  profile: 'No deployment profile set',
  tracking_domain: 'No tracking domain configured',
  ingress_schema: 'No ingress schema selected',
  default_currency: 'No default currency set',
  timezone: 'No timezone configured',
  edge_xdp: 'Edge XDP flag not reported',
  edge_expose_click: 'Click exposure flag not reported',
  edge_expose_openrtb: 'OpenRTB exposure flag not reported',
  network_interface: 'No network interface configured',
  click_url_template: 'Set a tracking domain to generate click URLs',
  openrtb_endpoint_template: 'Set a tracking domain to generate OpenRTB endpoints',
  stripe_secret_key: 'Stripe secret key not configured',
  stripe_webhook_secret: 'Stripe webhook secret not configured',
  stripe_enabled: 'Stripe billing flag not reported',
  stripe_checkout_success_url: 'No Stripe success redirect URL',
  stripe_checkout_cancel_url: 'No Stripe cancel redirect URL',
  telemetry_enabled: 'Telemetry flag not reported',
  license_valid_until: 'No expiry date on current license',
  license_deployment: 'Deployment ID not assigned yet',
  license_plan: 'No plan code in license metadata',
} as const;

export type SettingsEmptyField = keyof typeof EMPTY_COPY;

export function settingsEmptyMessage(field: SettingsEmptyField): string {
  return EMPTY_COPY[field];
}

export function settingsEmptyValue(field: SettingsEmptyField): ReactNode {
  return <span className="text-muted-foreground">{EMPTY_COPY[field]}</span>;
}

export function settingsTextValue(value: string, field: SettingsEmptyField): ReactNode {
  const trimmed = value.trim();
  if (trimmed) {
    if (usesSettingsDisplayValue(field)) {
      return formatSettingsDisplayValue(trimmed, field);
    }
    return trimmed;
  }
  return settingsEmptyValue(field);
}

export function settingsMonoValue(value: string, field: SettingsEmptyField): ReactNode {
  const trimmed = value.trim();
  if (trimmed) {
    return <span className="font-mono text-xs">{trimmed}</span>;
  }
  return settingsEmptyValue(field);
}

export function settingsBoolValue(
  value: boolean | undefined,
  field: SettingsEmptyField,
): ReactNode {
  if (value === undefined) {
    return settingsEmptyValue(field);
  }
  return value ? 'Enabled' : 'Disabled';
}

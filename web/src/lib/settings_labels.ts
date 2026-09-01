const SETTINGS_FIELD_LABELS: Record<string, string> = {
  bootstrap_complete: 'Initial setup complete',
  click_url_template: 'Click URL template',
  config: 'Configuration',
  default_currency: 'Default currency',
  edge_expose_click: 'Edge expose click',
  edge_expose_openrtb: 'Edge expose OpenRTB',
  edge_xdp: 'Edge XDP',
  ingress_schema: 'Ingress schema',
  network_interface: 'Network interface',
  openrtb_endpoint_template: 'OpenRTB endpoint template',
  profile: 'Deployment profile',
  restart_required: 'Restart required',
  secrets: 'Secrets',
  stripe: 'Stripe billing',
  stripe_checkout_cancel_url: 'Stripe cancel URL',
  stripe_checkout_success_url: 'Stripe success URL',
  stripe_enabled: 'Stripe',
  stripe_secret_key: 'Stripe secret key',
  stripe_webhook_secret: 'Stripe webhook secret',
  telemetry_enabled: 'Telemetry',
  timezone: 'Timezone',
  tracking_domain: 'Tracking domain',
};

export function settingsFieldLabel(key: string): string {
  const mapped = SETTINGS_FIELD_LABELS[key];
  if (mapped) {
    return mapped;
  }
  return key
    .split('_')
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

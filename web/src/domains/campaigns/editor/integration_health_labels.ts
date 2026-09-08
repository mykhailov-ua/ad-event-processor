const INTEGRATION_HEALTH_SLUG_LABELS: Record<string, string> = {
  browser_pixel_capi_dedup: 'Browser pixel / CAPI dedup',
  click_join_keys: 'Click join keys',
  cost_sync_credential: 'Cost sync credential',
  ingress_cost_config: 'Ingress cost config',
  integration_schema: 'Integration schema',
  postback_config: 'Postback config',
  target_url: 'Target URL',
  traffic_template: 'Traffic template',
};

function titleCaseFromSlug(slug: string): string {
  return slug
    .split('_')
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

export function formatIntegrationHealthSlug(slug: string): string {
  const normalized = slug.trim();
  if (normalized === '') {
    return 'Check';
  }
  return INTEGRATION_HEALTH_SLUG_LABELS[normalized] ?? titleCaseFromSlug(normalized);
}

export function formatIntegrationHealthStatus(status: string): string {
  switch (status.trim().toLowerCase()) {
    case 'ok':
      return 'OK';
    case 'warn':
      return 'Warning';
    case 'fail':
      return 'Failed';
    default:
      return titleCaseFromSlug(status);
  }
}

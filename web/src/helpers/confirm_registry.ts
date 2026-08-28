export type ConfirmLevel = 'none' | 'standard' | 'destructive' | 'financial' | 'strong' | 'retry';

export type ConfirmEntry = {
  level: ConfirmLevel;
  label?: string;
};

const registry = new Map<string, ConfirmEntry>([
  ['POST /auth/login', { level: 'none' }],
  ['POST /auth/logout', { level: 'standard', label: 'Exit?' }],
  ['POST /auth/register', { level: 'strong', label: 'Register user' }],
  ['POST /settings/platform/bootstrap', { level: 'strong', label: 'Initialize platform' }],
  ['PATCH /settings/platform', { level: 'standard' }],
  ['POST /settings/platform/apply', { level: 'destructive', label: 'Apply to disk' }],
  ['POST /selfserve/campaigns', { level: 'financial', label: 'Create campaign' }],
  ['PATCH /campaigns/{id}', { level: 'standard', label: 'Save campaign changes' }],
  ['PATCH /campaigns/{id}/fraud', { level: 'standard', label: 'Save fraud thresholds' }],
  ['POST /fraud/labels', { level: 'standard', label: 'Save fraud label' }],
  ['POST /fraud/labels/bulk', { level: 'standard', label: 'Import fraud labels' }],
  ['POST /fraud/overrides', { level: 'strong', label: 'Mark false positive' }],
  ['PATCH /ops/fraud/presets/{name}', { level: 'standard', label: 'Update fraud preset' }],
  [
    'POST /campaigns/{id}/placement-blocks',
    { level: 'destructive', label: 'Block placement / sub' },
  ],
  ['POST /selfserve/campaigns/{id}/pause', { level: 'destructive', label: 'Pause campaign' }],
  ['POST /selfserve/campaigns/{id}/resume', { level: 'standard', label: 'Resume campaign' }],
  ['POST /selfserve/payment-intents', { level: 'financial', label: 'Create payment' }],
  ['POST /selfserve/api-keys', { level: 'standard', label: 'Key shown once' }],
  ['POST /billing/invoices/preview', { level: 'none' }],
  ['POST /billing/invoices/{id}/void', { level: 'strong', label: 'Void invoice' }],
  ['POST /billing/invoices/{id}/deliveries/retry', { level: 'retry', label: 'Retry delivery' }],
  ['POST /billing/exports', { level: 'standard' }],
  ['PUT /customers/{id}/tax-profile', { level: 'standard' }],
  ['POST /cost-sync/credentials/{network}', { level: 'standard' }],
  ['PUT /cost-sync/credentials/{network}', { level: 'standard' }],
  ['DELETE /cost-sync/credentials/{network}', { level: 'destructive' }],
  ['POST /cost-sync/run', { level: 'standard' }],
  ['POST /brands', { level: 'standard' }],
  ['POST /brands/{id}/creatives', { level: 'standard' }],
  ['PATCH /brand-creatives/{id}', { level: 'standard' }],
  ['DELETE /brand-creatives/{id}', { level: 'destructive' }],
  ['POST /supply/sellers', { level: 'standard' }],
  ['PUT /supply/sellers/{id}', { level: 'standard' }],
  ['DELETE /supply/sellers/{id}', { level: 'destructive' }],
  ['POST /supply/ads-txt', { level: 'standard' }],
  ['PUT /supply/ads-txt/{id}', { level: 'standard' }],
  ['DELETE /supply/ads-txt/{id}', { level: 'destructive' }],
  ['PUT /postbacks/config/{campaign_id}', { level: 'standard' }],
  ['POST /postbacks/dlq/{id}/retry', { level: 'retry' }],
  ['POST /postbacks/config/{campaign_id}/test', { level: 'none' }],
  ['POST /team/members', { level: 'standard' }],
  ['PATCH /team/members/{id}', { level: 'standard' }],
  ['POST /team/budget-approvals/{id}/approve', { level: 'standard' }],
  ['POST /team/budget-approvals/{id}/deny', { level: 'destructive' }],
  ['PUT /campaigns/{id}/owner', { level: 'standard' }],
  ['POST /integration/templates/import', { level: 'standard' }],
  ['POST /campaigns/{id}/apply-templates', { level: 'standard' }],
  ['POST /margin-guard/policies', { level: 'standard' }],
  ['POST /margin-guard/overrides', { level: 'destructive' }],
  ['POST /smart-alerts/rules', { level: 'standard' }],
  ['PATCH /smart-alerts/rules/{id}', { level: 'standard' }],
  ['DELETE /smart-alerts/rules/{id}', { level: 'destructive' }],
  ['POST /smart-alerts/events/{id}/ack', { level: 'standard' }],
  ['POST /domains', { level: 'standard' }],
  ['DELETE /domains/{hostname}', { level: 'destructive' }],
  ['POST /domains/{hostname}/probe', { level: 'standard' }],
  ['POST /domains/{hostname}/ssl/setup', { level: 'strong', label: 'Setup SSL certificate' }],
  ['POST /domains/park', { level: 'strong', label: 'Park domain (Cloudflare DNS)' }],
  ['POST /ops/blacklist', { level: 'destructive', label: 'Block IP' }],
  ['DELETE /ops/blacklist', { level: 'destructive', label: 'Unblock IP' }],
  ['POST /ops/dlq/{id}/retry', { level: 'retry' }],
  ['POST /ops/dlq/inbox/{id}/retry', { level: 'retry' }],
  ['POST /ops/shards/0/catchup', { level: 'strong', label: 'Run shard 0 catch-up' }],
  ['POST /ops/fraud-threat', { level: 'destructive' }],
  ['POST /ops/ml-model/labels', { level: 'standard' }],
  ['POST /ops/roles/reload', { level: 'strong', label: 'Reload RBAC' }],
  ['POST /ops/support/bundle', { level: 'none' }],
  ['POST /rtb/deals', { level: 'standard' }],
  ['PATCH /rtb/deals/{id}', { level: 'standard' }],
  ['DELETE /rtb/deals/{id}', { level: 'strong' }],
  ['POST /rtb/floors/apply', { level: 'destructive', label: 'Apply floors' }],
  ['POST /rtb/validate-bid-request', { level: 'none' }],
  ['PUT /telegram/bots/{id}', { level: 'standard' }],
  ['POST /telegram/deeplink-tokens', { level: 'standard' }],
  ['POST /telegram/postbacks', { level: 'standard' }],
  ['PUT /telegram/postbacks/{id}', { level: 'standard' }],
  ['DELETE /telegram/postbacks/{id}', { level: 'destructive' }],
  ['POST /telegram/postbacks/{id}/test', { level: 'none' }],
  ['POST /views', { level: 'standard' }],
  ['PUT /views/{id}', { level: 'standard' }],
  ['DELETE /views/{id}', { level: 'destructive' }],
  ['POST /support/feedback', { level: 'none' }],
  ['POST /forecast/campaign', { level: 'none' }],
  [
    'POST /integration/templates/import',
    { level: 'standard', label: 'Import integration templates' },
  ],
  ['POST /integration/schemas', { level: 'standard', label: 'Create integration schema' }],
  [
    'POST /integration/schemas/{id}/apply',
    { level: 'standard', label: 'Apply integration schema' },
  ],
  [
    'POST /campaigns/{id}/apply-templates',
    { level: 'standard', label: 'Apply bundled integration templates' },
  ],
  [
    'POST /platform-campaigns/{campaign_id}/pause',
    { level: 'destructive', label: 'Pause campaign on ad platform' },
  ],
  [
    'POST /platform-campaigns/{campaign_id}/resume',
    { level: 'standard', label: 'Resume campaign on ad platform' },
  ],
  [
    'POST /platform-campaigns/{campaign_id}/budget',
    { level: 'financial', label: 'Set daily budget on ad platform' },
  ],
  [
    'DELETE /platform-campaigns/links/{campaign_id}/{network}',
    { level: 'destructive', label: 'Remove platform campaign link' },
  ],
]);

export function getConfirmLevel(method: string, path: string): ConfirmEntry {
  const key = `${method.toUpperCase()} ${path}`;
  const exact = registry.get(key);
  if (exact) return exact;

  for (const [pattern, entry] of registry) {
    if (matchesPattern(pattern, key)) return entry;
  }

  const meta = import.meta as ImportMeta & { env?: { DEV?: boolean; PROD?: boolean } };
  if (meta.env?.DEV && !meta.env?.PROD) {
    console.warn(`confirm_registry: no entry for "${key}"`);
  }
  return { level: 'standard' };
}

function matchesPattern(pattern: string, actual: string): boolean {
  const reStr = pattern.replace(/\{[^}]+\}/g, '[^/]+');
  return new RegExp(`^${reStr}$`).test(actual);
}

export { registry };

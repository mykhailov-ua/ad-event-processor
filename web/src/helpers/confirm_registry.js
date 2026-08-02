/**
 * @typedef {'none'|'standard'|'destructive'|'financial'|'strong'|'retry'} ConfirmLevel
 */

/**
 * @typedef {Object} ConfirmEntry
 * @property {ConfirmLevel} level
 * @property {string} [label]
 */

/** @type {Map<string, ConfirmEntry>} */
const registry = new Map([
  ['POST /auth/login', { level: 'none' }],
  ['POST /auth/logout', { level: 'standard', label: 'Exit?' }],
  ['POST /auth/register', { level: 'strong', label: 'Register user' }],
  ['POST /settings/platform/bootstrap', { level: 'strong', label: 'Initialize platform' }],
  ['PATCH /settings/platform', { level: 'standard' }],
  ['POST /settings/platform/apply', { level: 'destructive', label: 'Apply to disk' }],
  ['POST /selfserve/campaigns', { level: 'financial', label: 'Create campaign' }],
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
  ['DELETE /cost-sync/credentials/{network}', { level: 'destructive' }],
  ['POST /cost-sync/run', { level: 'standard' }],
  ['PUT /postbacks/config/{campaign_id}', { level: 'standard' }],
  ['POST /postbacks/dlq/{id}/retry', { level: 'retry' }],
  ['POST /margin-guard/policies', { level: 'standard' }],
  ['POST /margin-guard/overrides', { level: 'destructive' }],
  ['POST /ops/blacklist', { level: 'destructive', label: 'Block IP' }],
  ['DELETE /ops/blacklist', { level: 'destructive', label: 'Unblock IP' }],
  ['POST /ops/dlq/{id}/retry', { level: 'retry' }],
  ['POST /ops/fraud-threat', { level: 'destructive' }],
  ['POST /ops/ml-model/labels', { level: 'standard' }],
  ['POST /ops/roles/reload', { level: 'strong', label: 'Reload RBAC' }],
  ['POST /ops/support/bundle', { level: 'none' }],
  ['POST /rtb/deals', { level: 'standard' }],
  ['PATCH /rtb/deals/{id}', { level: 'standard' }],
  ['DELETE /rtb/deals/{id}', { level: 'strong' }],
  ['POST /rtb/floors/apply', { level: 'destructive', label: 'Apply floors' }],
  ['POST /rtb/validate-bid-request', { level: 'none' }],
  ['POST /views', { level: 'standard' }],
  ['PUT /views/{id}', { level: 'standard' }],
  ['DELETE /views/{id}', { level: 'destructive' }],
  ['POST /support/feedback', { level: 'none' }],
  ['POST /forecast/campaign', { level: 'none' }],
]);

/**
 * @param {string} method
 * @param {string} path - may include actual IDs matching {id} patterns
 * @returns {ConfirmEntry}
 */
export function getConfirmLevel(method, path) {
  const key = `${method.toUpperCase()} ${path}`;
  if (registry.has(key)) return registry.get(key);

  for (const [pattern, entry] of registry) {
    if (matchesPattern(pattern, key)) return entry;
  }

  if (import.meta.env?.DEV && !import.meta.env?.PROD) {
    console.warn(`confirm_registry: no entry for "${key}"`);
  }
  return { level: 'standard' };
}

/**
 * @param {string} pattern
 * @param {string} actual
 * @returns {boolean}
 */
function matchesPattern(pattern, actual) {
  const reStr = pattern.replace(/\{[^}]+\}/g, '[^/]+');
  return new RegExp(`^${reStr}$`).test(actual);
}

export { registry };

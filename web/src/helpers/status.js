/**
 * @param {string} status
 */
export function campaignStatusClass(status) {
  const v = (status || '').toLowerCase();
  if (v === 'active') return 'active';
  if (v === 'paused') return 'paused';
  if (v === 'archived') return 'archived';
  if (v === 'failed' || v === 'deleted') return 'failed';
  return 'pending';
}

/**
 * @param {string} status
 */
export function serviceStatusClass(status) {
  const v = (status || '').toLowerCase();
  if (v === 'ok' || v === 'healthy' || v === 'up' || v === 'pass') return 'active';
  if (v === 'degraded' || v === 'warning') return 'pending';
  return 'failed';
}

/**
 * @param {string} status
 */
export function invoiceStatusClass(status) {
  const v = (status || '').toLowerCase();
  if (v === 'paid' || v === 'finalized') return 'active';
  if (v === 'draft' || v === 'pending' || v === 'open') return 'pending';
  if (v === 'void' || v === 'voided' || v === 'failed') return 'failed';
  return 'pending';
}

/**
 * @param {string} status
 * @param {'campaign'|'service'|'invoice'} kind
 */
export function statusClassFor(status, kind) {
  if (kind === 'service') return serviceStatusClass(status);
  if (kind === 'invoice') return invoiceStatusClass(status);
  return campaignStatusClass(status);
}

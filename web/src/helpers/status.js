/**
 * Map a campaign status string to a status-badge modifier class.
 *
 * @param {string} status
 * @returns {'success'|'warning'|'neutral'|'critical'|'info'}
 */
export function campaignStatusClass(status) {
  const v = (status || '').toLowerCase();
  if (v === 'active') return 'success';
  if (v === 'paused') return 'warning';
  if (v === 'archived') return 'neutral';
  if (v === 'failed' || v === 'deleted') return 'critical';
  return 'info';
}

/**
 * Map a service health status string to a status-badge modifier class.
 *
 * @param {string} status
 * @returns {'success'|'warning'|'critical'}
 */
export function serviceStatusClass(status) {
  const v = (status || '').toLowerCase();
  if (v === 'ok' || v === 'healthy' || v === 'up' || v === 'pass') return 'success';
  if (v === 'degraded' || v === 'warning' || v === 'disabled') return 'warning';
  return 'critical';
}

/**
 * Map an invoice status string to a status-badge modifier class.
 *
 * @param {string} status
 * @returns {'success'|'info'|'critical'}
 */
export function invoiceStatusClass(status) {
  const v = (status || '').toLowerCase();
  if (v === 'paid' || v === 'finalized') return 'success';
  if (v === 'draft' || v === 'pending' || v === 'open') return 'info';
  if (v === 'void' || v === 'voided' || v === 'failed') return 'critical';
  return 'info';
}

/**
 * Resolve a status-badge modifier class for the given domain kind.
 *
 * @param {string} status
 * @param {'campaign'|'service'|'invoice'} kind
 * @returns {'success'|'warning'|'neutral'|'critical'|'info'}
 */
export function statusClassFor(status, kind) {
  if (kind === 'service') return serviceStatusClass(status);
  if (kind === 'invoice') return invoiceStatusClass(status);
  return campaignStatusClass(status);
}

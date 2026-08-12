export type StatusTone = 'success' | 'warning' | 'neutral' | 'critical' | 'info';

/**
 * Map a campaign status string to a status-badge modifier class.
 */
export function campaignStatusClass(status: string): StatusTone {
  const v = (status || '').toLowerCase();
  if (v === 'active') return 'success';
  if (v === 'paused') return 'warning';
  if (v === 'archived') return 'neutral';
  if (v === 'failed' || v === 'deleted') return 'critical';
  return 'info';
}

/**
 * Map a service health status string to a status-badge modifier class.
 */
export function serviceStatusClass(status: string): 'success' | 'warning' | 'critical' {
  const v = (status || '').toLowerCase();
  if (v === 'ok' || v === 'healthy' || v === 'up' || v === 'pass' || v === 'live') return 'success';
  if (v === 'degraded' || v === 'warning' || v === 'disabled' || v === 'planned') return 'warning';
  return 'critical';
}

/**
 * Map an invoice status string to a status-badge modifier class.
 */
export function invoiceStatusClass(status: string): 'success' | 'info' | 'critical' {
  const v = (status || '').toLowerCase();
  if (v === 'paid' || v === 'finalized') return 'success';
  if (v === 'draft' || v === 'pending' || v === 'open') return 'info';
  if (v === 'void' || v === 'voided' || v === 'failed') return 'critical';
  return 'info';
}

/**
 * Resolve a status-badge modifier class for the given domain kind.
 */
export function statusClassFor(
  status: string,
  kind: 'campaign' | 'service' | 'invoice',
): StatusTone {
  if (kind === 'service') return serviceStatusClass(status);
  if (kind === 'invoice') return invoiceStatusClass(status);
  return campaignStatusClass(status);
}

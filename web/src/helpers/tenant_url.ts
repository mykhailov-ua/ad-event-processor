/**
 * Build shareable report query string with tenant customer_id for admins.
 */
export function tenantReportQueryString(params: Record<string, string>): string {
  const qs = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v) qs.set(k, v);
  }
  return qs.toString();
}

/**
 * Merge customer_id into current location for admin share links.
 */
export function syncTenantCustomerToUrl(customerId: string): void {
  if (!customerId) return;
  try {
    const url = new URL(window.location.href);
    if (url.searchParams.get('customer_id') === customerId) return;
    url.searchParams.set('customer_id', customerId);
    window.history.replaceState(null, '', url.pathname + url.search);
  } catch {
    // ignore
  }
}
